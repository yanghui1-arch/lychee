import { useEffect, useReducer, useState } from 'react'
import { translate, type TranslationRequest } from './translation-api'

export const INPUT_IDLE_MS = 600
type TranslationState = {
  status: 'idle' | 'waiting' | 'loading' | 'success' | 'error'
  text: string
  error: string
}
const idle: TranslationState = { status: 'idle', text: '', error: '' }

export function useTranslation(
  request: TranslationRequest,
  enabled: boolean,
  composing: boolean,
) {
  const { text, source_language, target_language } = request
  const [state, setState] = useState(idle)
  const [attempt, retry] = useReducer((value: number) => value + 1, 0)

  useEffect(() => {
    if (!enabled || !text.trim()) {
      setState(idle)
      return
    }
    setState({ status: 'waiting', text: '', error: '' })
    if (composing) return

    let active = true
    const controller = new AbortController()
    const timer = window.setTimeout(async () => {
      setState({ status: 'loading', text: '', error: '' })
      try {
        const result = await translate(
          { text, source_language, target_language },
          controller.signal,
        )
        if (active) setState({ status: 'success', text: result, error: '' })
      } catch (error) {
        if (active) {
          setState({
            status: 'error',
            text: '',
            error:
              error instanceof TypeError
                ? '无法连接翻译服务，请检查连接后重试。'
                : error instanceof Error
                  ? error.message
                  : '翻译失败，请重试。',
          })
        }
      }
    }, INPUT_IDLE_MS)

    return () => {
      active = false
      window.clearTimeout(timer)
      controller.abort()
    }
  }, [text, source_language, target_language, enabled, composing, attempt])

  return { ...state, retry }
}
