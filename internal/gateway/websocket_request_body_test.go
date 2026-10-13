package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"gpt-load/internal/channel"
	"gpt-load/internal/execution"
	"gpt-load/internal/state"
	"gpt-load/internal/telemetry"
)

func holdFirstWebsocketOpen(t *testing.T, handler *Handler) (<-chan struct{}, func()) {
	t.Helper()
	held := make(chan struct{})
	release := make(chan struct{})
	var started sync.Once
	var stop sync.Once
	handler.forwarder = websocketScriptForwarder{AttemptForwarder: handler.forwarder, open: func(context.Context, ForwardInput) (execution.WebsocketSession, execution.WebsocketResult) {
		started.Do(func() { close(held) })
		<-release
		session := &websocketScriptSession{done: make(chan struct{})}
		session.turn = func(context.Context, []byte, func(context.Context, []byte) error) execution.WebsocketResult {
			return execution.WebsocketResult{DispatchState: execution.DispatchMaybeSent, Error: &execution.ErrorEvidence{Kind: execution.ErrorKindCanceled}}
		}
		return session, execution.WebsocketResult{DispatchState: execution.DispatchNotSent}
	}}
	return held, func() { stop.Do(func() { close(release) }) }
}

func publishWebsocketRecording(t *testing.T, handler *Handler, input state.CompileInput, enabled bool) {
	t.Helper()
	input.AccessKeys[0].RecordRequestBody = enabled
	if _, err := handler.manager.Publish(input); err != nil {
		t.Fatal(err)
	}
}

func websocketLogByCode(events []telemetry.RequestEvent, code string) (telemetry.RequestEvent, bool) {
	for _, event := range events {
		if event.ErrorCode == code {
			return event, true
		}
	}
	return telemetry.RequestEvent{}, false
}

func websocketLogByModel(events []telemetry.RequestEvent, model string) (telemetry.RequestEvent, bool) {
	for _, event := range events {
		if event.ClientModel == model {
			return event, true
		}
	}
	return telemetry.RequestEvent{}, false
}

func TestWebsocketDroppedRequestBodyOnStreamLimit(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				for {
					_, message, err := conn.ReadMessage()
					if err != nil {
						return
					}
					var frame struct {
						StreamID string `json:"stream_id"`
					}
					_ = json.Unmarshal(message, &frame)
					if conn.WriteMessage(websocket.TextMessage, websocketCompleted("resp_limit", frame.StreamID)) != nil {
						return
					}
				}
			}))
			defer upstream.Close()
			handler, engine, input := websocketTestHandler(t, upstream.URL, channel.OpenAI)
			handler.websocketLimits.lanes = 1
			publishWebsocketRecording(t, handler, input, enabled)
			sink := &recordingRequestLogSink{}
			handler.requestLogSink = sink
			server := httptest.NewServer(engine)
			defer server.Close()
			conn := dialGatewayWebsocket(t, server.URL)

			kept := `{"type":"response.create","model":"public","input":"kept","stream_id":"lane-a"}`
			if err := conn.WriteMessage(websocket.TextMessage, []byte(kept)); err != nil {
				t.Fatal(err)
			}
			if _, _, err := conn.ReadMessage(); err != nil {
				t.Fatal(err)
			}
			rejected := `{"type":"response.create","model":"public","input":"limit-secret","stream_id":"lane-b"}`
			if err := conn.WriteMessage(websocket.TextMessage, []byte(rejected)); err != nil {
				t.Fatal(err)
			}
			var response struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := conn.ReadJSON(&response); err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != "websocket_stream_limit_reached" {
				t.Fatalf("limit response = %+v", response)
			}
			logs := waitWebsocketLogs(t, sink, 2)
			event, ok := websocketLogByCode(logs, "websocket_stream_limit_reached")
			if !ok || event.Status != telemetry.RequestStatusError {
				t.Fatalf("limit log missing: %+v", logs)
			}
			if enabled && string(event.RequestBody) != rejected {
				t.Fatalf("recorded body = %q, want rejected client frame", event.RequestBody)
			}
			if !enabled && event.RequestBody != nil {
				t.Fatalf("recording disabled logged body %q", event.RequestBody)
			}
		})
	}
}

func TestWebsocketDroppedRequestBodyOnQueuedCancel(t *testing.T) {
	handler, engine, input := websocketTestHandler(t, "http://127.0.0.1:1", channel.OpenAI)
	handler.websocketLimits.active = 1
	handler.websocketLimits.lanes = 1
	publishWebsocketRecording(t, handler, input, true)
	sink := &recordingRequestLogSink{}
	handler.requestLogSink = sink
	held, release := holdFirstWebsocketOpen(t, handler)
	server := httptest.NewServer(engine)
	defer server.Close()
	defer release()
	conn := dialGatewayWebsocket(t, server.URL)

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"public","input":"held","stream_id":"lane-a"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-held:
	case <-time.After(5 * time.Second):
		t.Fatal("first turn did not reach upstream open")
	}
	queued := `{"type":"response.create","model":"queued-marker","input":"queued-secret","stream_id":"lane-a"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(queued)); err != nil {
		t.Fatal(err)
	}
	rejected := `{"type":"response.create","model":"public","input":"limit-secret","stream_id":"lane-b"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(rejected)); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "websocket_stream_limit_reached" {
		t.Fatalf("queue barrier = %+v", response)
	}
	_ = conn.Close()
	logs := waitWebsocketLogs(t, sink, 2)
	event, ok := websocketLogByModel(logs, "queued-marker")
	if !ok || event.Status != telemetry.RequestStatusCanceled {
		t.Fatalf("queued cancel log = %+v", logs)
	}
	if string(event.RequestBody) != queued {
		t.Fatalf("recorded body = %q, want queued client frame", event.RequestBody)
	}
}

func TestWebsocketDroppedRequestBodyOnExecuteTurnCancel(t *testing.T) {
	handler, engine, input := websocketTestHandler(t, "http://127.0.0.1:1", channel.OpenAI)
	publishWebsocketRecording(t, handler, input, true)
	sink := &recordingRequestLogSink{}
	handler.requestLogSink = sink
	var ids atomic.Int32
	startedSecond := make(chan struct{})
	handler.newRequestID = func() (string, error) {
		n := ids.Add(1)
		if n == 2 {
			close(startedSecond)
		}
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", n), nil
	}
	held, release := holdFirstWebsocketOpen(t, handler)
	server := httptest.NewServer(engine)
	defer server.Close()
	defer release()
	conn := dialGatewayWebsocket(t, server.URL)

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"public","input":"held","stream_id":"lane-a"}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-held:
	case <-time.After(5 * time.Second):
		t.Fatal("first turn did not reach upstream open")
	}
	canceled := `{"type":"response.create","model":"cancel-marker","input":"cancel-secret","stream_id":"lane-b"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(canceled)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-startedSecond:
	case <-time.After(5 * time.Second):
		t.Fatalf("second turn did not enter executeTurn, ids=%d", ids.Load())
	}
	_ = conn.Close()
	logs := waitWebsocketLogs(t, sink, 1)
	event, ok := websocketLogByModel(logs, "cancel-marker")
	if !ok || event.Status != telemetry.RequestStatusCanceled {
		t.Fatalf("executeTurn cancel log = %+v", logs)
	}
	if string(event.RequestBody) != canceled {
		t.Fatalf("recorded body = %q, want canceled client frame", event.RequestBody)
	}
}

func TestWebsocketDroppedRequestBodyOmitsUnauthorized(t *testing.T) {
	t.Run("queued", func(t *testing.T) {
		handler, engine, input := websocketTestHandler(t, "http://127.0.0.1:1", channel.OpenAI)
		handler.websocketLimits.active = 1
		handler.websocketLimits.lanes = 1
		publishWebsocketRecording(t, handler, input, true)
		sink := &recordingRequestLogSink{}
		handler.requestLogSink = sink
		held, release := holdFirstWebsocketOpen(t, handler)
		server := httptest.NewServer(engine)
		defer server.Close()
		defer release()
		conn := dialGatewayWebsocket(t, server.URL)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"public","input":"held","stream_id":"lane-a"}`)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-held:
		case <-time.After(5 * time.Second):
			t.Fatal("first turn did not reach upstream open")
		}
		queued := `{"type":"response.create","model":"queued-marker","input":"queued-secret","stream_id":"lane-a"}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(queued)); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"public","input":"limit-secret","stream_id":"lane-b"}`)); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := conn.ReadJSON(&response); err != nil {
			t.Fatal(err)
		}
		if response.Error.Code != "websocket_stream_limit_reached" {
			t.Fatalf("queue barrier = %+v", response)
		}
		input.AccessKeys[0].AllowedPeerCIDRs = []netip.Prefix{netip.MustParsePrefix("203.0.113.5/32")}
		if _, err := handler.manager.Publish(input); err != nil {
			t.Fatal(err)
		}
		logs := waitWebsocketLogs(t, sink, 2)
		event, ok := websocketLogByModel(logs, "queued-marker")
		if !ok || event.Status != telemetry.RequestStatusCanceled {
			t.Fatalf("queued cancel log = %+v", logs)
		}
		if event.RequestBody != nil {
			t.Fatalf("unauthorized queued log recorded body %q", event.RequestBody)
		}
	})

	t.Run("executeTurn", func(t *testing.T) {
		handler, engine, input := websocketTestHandler(t, "http://127.0.0.1:1", channel.OpenAI)
		publishWebsocketRecording(t, handler, input, true)
		sink := &recordingRequestLogSink{}
		handler.requestLogSink = sink
		var ids atomic.Int32
		startedSecond := make(chan struct{})
		handler.newRequestID = func() (string, error) {
			n := ids.Add(1)
			if n == 2 {
				close(startedSecond)
			}
			return fmt.Sprintf("00000000-0000-4000-8000-%012d", n), nil
		}
		held, release := holdFirstWebsocketOpen(t, handler)
		server := httptest.NewServer(engine)
		defer server.Close()
		defer release()
		conn := dialGatewayWebsocket(t, server.URL)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"response.create","model":"public","input":"held","stream_id":"lane-a"}`)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-held:
		case <-time.After(5 * time.Second):
			t.Fatal("first turn did not reach upstream open")
		}
		canceled := `{"type":"response.create","model":"cancel-marker","input":"cancel-secret","stream_id":"lane-b"}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(canceled)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-startedSecond:
		case <-time.After(5 * time.Second):
			t.Fatalf("second turn did not enter executeTurn, ids=%d", ids.Load())
		}
		input.AccessKeys[0].AllowedPeerCIDRs = []netip.Prefix{netip.MustParsePrefix("203.0.113.5/32")}
		if _, err := handler.manager.Publish(input); err != nil {
			t.Fatal(err)
		}
		logs := waitWebsocketLogs(t, sink, 1)
		event, ok := websocketLogByModel(logs, "cancel-marker")
		if !ok || event.Status != telemetry.RequestStatusCanceled {
			t.Fatalf("executeTurn cancel log = %+v", logs)
		}
		if event.RequestBody != nil {
			t.Fatalf("unauthorized executeTurn log recorded body %q", event.RequestBody)
		}
	})
}
