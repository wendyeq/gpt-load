import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { generationTokensPerSecond, outputTokensPerSecond } from './output-rate.ts'

const sample = {
  protocol: 'openai-completions',
  status_code: 200,
  usage_state: 'complete',
  stream: true,
  output_tokens: '156',
  duration_ms: 6400,
  first_response_ms: 1500 as number | null,
}

function shown(rate: number | null): string {
  return rate === null
    ? '—'
    : new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 }).format(rate)
}

describe('request log throughput', () => {
  it('156 tokens, 6400ms, first response 1500ms → 24.4 and 31.8', () => {
    assert.equal(shown(outputTokensPerSecond(sample)), '24.4')
    assert.equal(shown(generationTokensPerSecond(sample)), '31.8')
  })

  it('returns null without a first response', () => {
    assert.equal(generationTokensPerSecond({ ...sample, stream: false }), null)
    assert.equal(generationTokensPerSecond({ ...sample, first_response_ms: null }), null)
    assert.equal(generationTokensPerSecond({ ...sample, first_response_ms: 0 }), null)
  })

  it('returns null when output is zero', () => {
    assert.equal(generationTokensPerSecond({ ...sample, output_tokens: '0' }), null)
  })

  it('returns null when first response is not earlier than the total duration', () => {
    assert.equal(generationTokensPerSecond({ ...sample, first_response_ms: 6400 }), null)
    assert.equal(generationTokensPerSecond({ ...sample, first_response_ms: 6401 }), null)
  })

  it('returns null when the generation window is under 500ms', () => {
    assert.equal(
      generationTokensPerSecond({ ...sample, duration_ms: 1999, first_response_ms: 1500 }),
      null,
    )
  })

  it('keeps the rate at the 500ms cutoff', () => {
    assert.equal(
      generationTokensPerSecond({ ...sample, duration_ms: 2000, first_response_ms: 1500 }),
      312,
    )
  })
})
