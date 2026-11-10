import { useLayoutEffect, useRef } from 'react'

export function ListeningSourceLabel({
  label,
  hasSound,
}: {
  label: string
  hasSound: boolean
}) {
  const viewport = useRef<HTMLSpanElement>(null)
  const track = useRef<HTMLSpanElement>(null)
  const animation = useRef<Animation | null>(null)

  useLayoutEffect(() => {
    const container = viewport.current!
    const content = track.current!
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    let previousDistance = -1
    const measure = () => {
      const distance = Math.max(0, content.scrollWidth - container.clientWidth)
      if (distance === previousDistance) return
      previousDistance = distance
      animation.current?.cancel()
      animation.current = null
      if (distance <= 1 || reducedMotion.matches) return

      // Reset while fully transparent so the loop has no visible position jump.
      const pause = 1500
      const scrollEnd = pause + (distance / 28) * 1000
      const fadeStart = scrollEnd + pause
      const fadeEnd = fadeStart + 450
      const resetEnd = fadeEnd + 80
      const duration = resetEnd + 550
      const endTransform = `translateX(-${distance}px)`
      animation.current = content.animate(
        [
          { transform: 'translateX(0)', opacity: 1, offset: 0 },
          {
            transform: 'translateX(0)',
            opacity: 1,
            offset: pause / duration,
          },
          {
            transform: endTransform,
            opacity: 1,
            offset: scrollEnd / duration,
          },
          {
            transform: endTransform,
            opacity: 1,
            offset: fadeStart / duration,
            easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
          },
          {
            transform: endTransform,
            opacity: 0,
            offset: fadeEnd / duration,
          },
          {
            transform: 'translateX(0)',
            opacity: 0,
            offset: resetEnd / duration,
            easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
          },
          { transform: 'translateX(0)', opacity: 1, offset: 1 },
        ],
        { duration, iterations: Infinity, easing: 'linear' },
      )
    }
    const observer = new ResizeObserver(measure)
    observer.observe(container)
    observer.observe(content)
    const updateMotion = () => {
      previousDistance = -1
      measure()
    }
    reducedMotion.addEventListener('change', updateMotion)
    measure()
    return () => {
      observer.disconnect()
      reducedMotion.removeEventListener('change', updateMotion)
      animation.current?.cancel()
      animation.current = null
    }
  }, [label])

  return (
    <span
      ref={viewport}
      className="block min-w-0 overflow-hidden whitespace-nowrap"
      title={`正在监听 ${label}`}
      onPointerEnter={() => {
        const current = animation.current
        if (!current) return
        // Hover should always leave readable text, even during the fade.
        if (Number(getComputedStyle(track.current!).opacity) < 0.99) {
          current.currentTime = 0
        }
        current.pause()
      }}
      onPointerLeave={() => animation.current?.play()}
    >
      <span
        ref={track}
        className="inline-flex w-max [vertical-align:top] [@media(prefers-reduced-motion:reduce)]:block [@media(prefers-reduced-motion:reduce)]:max-w-full [@media(prefers-reduced-motion:reduce)]:overflow-hidden [@media(prefers-reduced-motion:reduce)]:text-ellipsis"
      >
        <span>正在监听 {label} ·&nbsp;</span>
        <span className="w-[5em]">{hasSound ? '检测到声音' : '等待声音'}</span>
      </span>
    </span>
  )
}
