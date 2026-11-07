import { useEffect, useLayoutEffect, useRef } from 'react'
import { gsap } from 'gsap'
import { SetFloatingExpanded } from '../../wailsjs/go/app/WindowService'

export function useFloatingControlsAnimation(hovered: boolean) {
  const rootRef = useRef<HTMLDivElement>(null)
  const drawerRef = useRef<HTMLDivElement>(null)
  const closeRef = useRef<HTMLButtonElement>(null)
  const timelineRef = useRef<gsap.core.Timeline | null>(null)
  const revisionRef = useRef(0)

  useLayoutEffect(() => {
    const drawer = drawerRef.current
    const close = closeRef.current
    if (!drawer || !close) return

    const context = gsap.context(() => {
      // The fixed viewport hides the drawer completely behind the ball.
      // Animate its full width out of that pocket, without fading or scaling.
      gsap.set(drawer, {
        autoAlpha: 1,
        xPercent: 100,
        force3D: true,
      })
      gsap.set(close, {
        autoAlpha: 0,
        scale: 0.8,
        transformOrigin: 'center',
        force3D: true,
      })
      timelineRef.current = gsap
        .timeline({ paused: true })
        .to(drawer, { xPercent: 0, duration: 0.32, ease: 'power2.out' }, 0)
        .to(
          close,
          { autoAlpha: 1, scale: 1, duration: 0.18, ease: 'power2.out' },
          0.14,
        )
    }, rootRef)

    return () => {
      timelineRef.current = null
      context.revert()
    }
  }, [])

  useEffect(() => {
    const timeline = timelineRef.current
    if (!timeline) return
    const revision = ++revisionRef.current
    const reducedMotion = window.matchMedia(
      '(prefers-reduced-motion: reduce)',
    ).matches
    let cancelled = false

    const collapse = () => {
      if (!cancelled) {
        void SetFloatingExpanded(false, revision).catch(console.error)
      }
    }

    timeline.eventCallback('onReverseComplete', null)
    if (hovered) {
      // Open the native clip before drawing the controls. If a leave arrives
      // while this call is pending, it must not restart the entrance animation.
      timeline.pause()
      void SetFloatingExpanded(true, revision)
        .then(() => {
          if (cancelled) return
          if (reducedMotion) timeline.progress(1, true).pause()
          else timeline.timeScale(1).play()
        })
        .catch(console.error)
    } else {
      timeline.eventCallback('onReverseComplete', collapse)
      if (reducedMotion || timeline.progress() === 0) {
        timeline.pause(0, true)
        collapse()
      } else {
        // Reverse the same playhead, preserving the current visual position.
        // The native hit area is released only when both controls are hidden.
        timeline.timeScale(1.2).reverse()
      }
    }

    return () => {
      cancelled = true
      timeline.eventCallback('onReverseComplete', null)
    }
  }, [hovered])

  return { rootRef, drawerRef, closeRef }
}
