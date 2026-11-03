type OutputTiming = {
  protocol: string
  status_code: number
  duration_ms: number
  output_tokens: string
  usage_state: string
}

type GenerationTiming = {
  stream: boolean
  first_response_ms: number | null
  duration_ms: number
  output_tokens: string
}

// 生成窗口短于 0.5s 时速度不稳定，返回 null。
const minimumGenerationWindowMs = 500

// 两套前端共用请求平均输出速度，包含等待、重试、凭据轮换及思考耗时。
export function outputTokensPerSecond(row: OutputTiming): number | null {
  const tokens = Number(row.output_tokens)
  if (
    row.status_code < 200 ||
    row.status_code >= 300 ||
    !['openai-completions', 'openai-responses', 'anthropic', 'gemini'].includes(row.protocol) ||
    (row.usage_state !== 'complete' && row.usage_state !== 'partial') ||
    !Number.isSafeInteger(tokens) ||
    tokens <= 0
  ) {
    return null
  }
  return Number.isSafeInteger(row.duration_ms) && row.duration_ms > 0
    ? tokens / (row.duration_ms / 1000)
    : null
}

export function generationTokensPerSecond(row: GenerationTiming): number | null {
  const tokens = Number(row.output_tokens)
  const first = row.first_response_ms
  if (
    !row.stream ||
    first === null ||
    !Number.isSafeInteger(first) ||
    first <= 0 ||
    !Number.isSafeInteger(tokens) ||
    tokens <= 0 ||
    !Number.isSafeInteger(row.duration_ms) ||
    row.duration_ms - first < minimumGenerationWindowMs
  ) {
    return null
  }
  return tokens / ((row.duration_ms - first) / 1000)
}
