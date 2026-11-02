package control

import "testing"

func TestAntigravityClassifyModelKeepsArgonOutOfGeminiPool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		model string
		want  antigravityQuotaPool
	}{
		{model: "gemini-2.5-flash", want: antigravityPoolGemini},
		{model: "gemini-3-pro", want: antigravityPoolGemini},
		{model: "Gemini-Flash", want: antigravityPoolGemini},
		{model: "gemini-argon", want: antigravityPoolArgon},
		{model: "gemini-4-argon-preview", want: antigravityPoolArgon},
		{model: "claude-sonnet-4-5", want: antigravityPoolClaudeGPT},
		{model: "gpt-4.1", want: antigravityPoolClaudeGPT},
		{model: "chatgpt-4o", want: antigravityPoolClaudeGPT},
		{model: "o3-mini", want: antigravityPoolClaudeGPT},
		{model: "unknown-router", want: antigravityPoolUnknown},
		{model: "", want: antigravityPoolUnknown},
	}
	for _, test := range tests {
		if got := antigravityClassifyModel(test.model); got != test.want {
			t.Errorf("antigravityClassifyModel(%q) = %d, want %d", test.model, got, test.want)
		}
	}
}

func TestAntigravityModelBelongsToWindowUsesPoolIdentity(t *testing.T) {
	t.Parallel()
	weekly := antigravityWeeklyWindowSeconds
	gemini := ObservationQuotaWindow{
		ID: "gemini-weekly", Label: "Gemini · 7d", Scope: "model", WindowSeconds: &weekly,
	}
	argon := ObservationQuotaWindow{
		ID: "gemini-argon-wtu-weekly", Label: "Gemini 4 Argon · 7d", Scope: "model", WindowSeconds: &weekly,
	}
	claude := ObservationQuotaWindow{
		ID: "3p-weekly", Label: "Claude/GPT · 7d", Scope: "model", WindowSeconds: &weekly,
	}
	unknown := ObservationQuotaWindow{
		ID: "custom-weekly", Label: "Custom API Pool · 7d", Scope: "model", WindowSeconds: &weekly,
	}
	tests := []struct {
		model  string
		window ObservationQuotaWindow
		want   bool
	}{
		{model: "gemini-3-pro", window: gemini, want: true},
		{model: "gemini-argon", window: gemini, want: false},
		{model: "gemini-argon", window: argon, want: true},
		{model: "gemini-3-pro", window: argon, want: false},
		{model: "claude-sonnet-4-5", window: claude, want: true},
		{model: "gemini-2.5-flash", window: claude, want: false},
		{model: "gemini-3-pro", window: unknown, want: false},
		{model: "mystery", window: gemini, want: false},
	}
	for _, test := range tests {
		if got := antigravityModelBelongsToWindow(test.model, test.window); got != test.want {
			t.Errorf(
				"antigravityModelBelongsToWindow(%q, %s) = %t, want %t",
				test.model, test.window.ID, got, test.want,
			)
		}
	}
}

func TestAntigravityWeeklyModelWindowIgnoresFiveHourAndAccount(t *testing.T) {
	t.Parallel()
	weekly := antigravityWeeklyWindowSeconds
	fiveHour := int64(5 * 60 * 60)
	if !antigravityWeeklyModelWindow(ObservationQuotaWindow{
		Scope: "model", WindowSeconds: &weekly,
	}) {
		t.Fatal("weekly model window should be included")
	}
	if antigravityWeeklyModelWindow(ObservationQuotaWindow{
		Scope: "model", WindowSeconds: &fiveHour,
	}) {
		t.Fatal("5h model window should not be included")
	}
	if antigravityWeeklyModelWindow(ObservationQuotaWindow{
		Scope: "account", WindowSeconds: &weekly,
	}) {
		t.Fatal("account weekly window should stay on the account path")
	}
}
