import { useLayoutEffect, useRef } from 'react'
import type { Page } from './model'

export function useTranslationPanelHeight(
  text: string,
  page: Page,
  textSize: number,
) {
  const panelsRef = useRef<HTMLDivElement>(null)
  const sourceRef = useRef<HTMLTextAreaElement>(null)

  useLayoutEffect(() => {
    const panels = panelsRef.current
    const source = sourceRef.current
    const body = panels?.parentElement
    if (page !== 'translate' || !panels || !source || !body) return

    const resize = () => {
      if (window.matchMedia('(max-width: 760px)').matches) {
        panels.style.removeProperty('--translation-panel-height')
        return
      }
      // Measure a hidden copy so typing keeps its caret and scroll position.
      const measure = source.cloneNode() as HTMLTextAreaElement
      measure.value = text
      measure.removeAttribute('id')
      measure.setAttribute('aria-hidden', 'true')
      measure.tabIndex = -1
      Object.assign(measure.style, {
        position: 'absolute',
        width: `${source.clientWidth}px`,
        height: '0',
        minHeight: '0',
        visibility: 'hidden',
        overflow: 'hidden',
        pointerEvents: 'none',
      })
      source.parentElement!.append(measure)
      const contentHeight = measure.scrollHeight
      measure.remove()

      const chromeHeight = panels.offsetHeight - source.offsetHeight
      const height = Math.min(
        body.clientHeight * 0.9,
        Math.max(body.clientHeight * 0.5, contentHeight + chromeHeight),
      )
      panels.style.setProperty('--translation-panel-height', `${height}px`)
    }

    resize()
    const observer = new ResizeObserver(resize)
    observer.observe(body)
    document.fonts.addEventListener('loadingdone', resize)
    return () => {
      observer.disconnect()
      document.fonts.removeEventListener('loadingdone', resize)
      panels.style.removeProperty('--translation-panel-height')
    }
  }, [text, page, textSize])

  return { panelsRef, sourceRef }
}
