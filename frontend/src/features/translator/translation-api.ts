import type { Language, SourceLanguage } from './model'

export type TranslationRequest = {
  text: string
  target_language: Language
  source_language: SourceLanguage
}

export async function translate(
  request: TranslationRequest,
  signal: AbortSignal,
): Promise<string> {
  const response = await fetch('/api/translate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
    signal,
  })
  const body: unknown = await response.json().catch(() => null)
  const data = body as {
    translated_text?: unknown
    error?: { message?: unknown }
  } | null
  if (!response.ok) {
    throw new Error(
      typeof data?.error?.message === 'string'
        ? data.error.message
        : `翻译服务暂时不可用（${response.status}），请重试。`,
    )
  }
  if (typeof data?.translated_text !== 'string') {
    throw new Error('翻译服务返回了无效结果，请重试。')
  }
  return data.translated_text
}
