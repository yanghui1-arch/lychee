import { useEffect, useReducer, useRef, useState } from 'react'
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
  const requestController = useRef<AbortController | null>(null)

  // Tab changes keep the current request and result; input changes invalidate them.
  useEffect(() => {
    return () => {
      requestController.current?.abort()
      requestController.current = null
    }
  }, [text, source_language, target_language, composing, attempt])

  useEffect(() => {
    if (!text.trim()) {
      setState(idle)
      return
    }
    if (!enabled || requestController.current) return
    setState({ status: 'waiting', text: '', error: '' })
    if (composing) return

    const timer = window.setTimeout(async () => {
      const controller = new AbortController()
      requestController.current = controller
      setState({ status: 'loading', text: '', error: '' })
      try {
        const result = await translate(
          { text, source_language, target_language },
          controller.signal,
        )
        if (!controller.signal.aborted)
          setState({ status: 'success', text: result, error: '' })
      } catch (error) {
        if (!controller.signal.aborted) {
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
      window.clearTimeout(timer)
    }
  }, [text, source_language, target_language, enabled, composing, attempt])

  return { ...(enabled ? state : idle), retry }
}
