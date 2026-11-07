import { useEffect, useRef, useState } from 'react'
import { Scissors, X } from 'lucide-react'
import floatingIcon from '@/assets/lychee-floating.svg?raw'
import { closeApp } from '@/lib/desktop'
import { cn } from '@/lib/utils'
import { useAppearance } from './useAppearance'
import { useFloatingControlsAnimation } from './useFloatingControlsAnimation'
import { OpenWorkspace, ShowFloating } from '../../wailsjs/go/app/WindowService'

export function FloatingBall({ onOpen }: { onOpen: () => void }) {
  useAppearance()
  const opening = useRef(false)
  const [error, setError] = useState(false)
  const [hovered, setHovered] = useState(false)
  const [dimmed, setDimmed] = useState(false)
  const { rootRef, drawerRef, closeRef } = useFloatingControlsAnimation(hovered)

  useEffect(() => {
    // Reveal only after the inline SVG has reached the compositor. The backend
    // also makes this idempotent for React StrictMode and delayed callbacks.
    let frame = requestAnimationFrame(() => {
      frame = requestAnimationFrame(() => {
        void ShowFloating().catch(console.error)
      })
    })
    return () => cancelAnimationFrame(frame)
  }, [])

  useEffect(() => {
    if (hovered) return
    const timer = window.setTimeout(() => setDimmed(true), 3000)
    return () => window.clearTimeout(timer)
  }, [hovered])

  function reveal() {
    setHovered(true)
    setDimmed(false)
  }

  async function open() {
    if (opening.current) return
    opening.current = true
    try {
      await OpenWorkspace()
      onOpen()
    } catch (cause) {
      console.error(cause)
      opening.current = false
      setError(true)
    }
  }

  return (
    <div
      ref={rootRef}
      className="fixed inset-0 select-none overflow-hidden"
      onPointerEnter={reveal}
      onPointerLeave={() => setHovered(false)}
      onFocusCapture={reveal}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) {
          setHovered(false)
        }
      }}
    >
      <div className="pointer-events-none absolute right-7 bottom-2 z-0 h-10 w-[84px] overflow-hidden [mask-image:radial-gradient(circle_at_right_center,transparent_23.5px,#000_24px)]">
        <div
          ref={drawerRef}
          className={cn(
            'invisible absolute inset-0 flex items-center rounded-l-[10px] border border-border bg-card text-foreground will-change-transform',
            hovered ? 'pointer-events-auto' : 'pointer-events-none',
          )}
          role="toolbar"
          aria-label="悬浮球工具"
          aria-hidden={!hovered}
          inert={!hovered}
        >
          <div className="flex h-full w-12 shrink-0 items-center justify-center">
            <button
              type="button"
              className="flex size-8 cursor-default items-center justify-center rounded-[6px] bg-transparent outline-none transition-colors duration-150 hover:text-[#d64f6b] focus-visible:text-[#d64f6b] focus-visible:ring-2 focus-visible:ring-ring/70"
              aria-label="截图翻译"
              aria-disabled="true"
              title="截图翻译"
            >
              <Scissors
                className="pointer-events-none size-[18px]"
                strokeWidth={1.8}
              />
            </button>
          </div>
        </div>
      </div>
      <button
        type="button"
        className="absolute right-0 bottom-0 z-10 block size-14 cursor-grab appearance-none overflow-hidden rounded-full border-0 bg-transparent p-0 outline-none [--wails-draggable:drag]! active:cursor-grabbing focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[#B46F5B]"
        aria-label="双击打开荔枝"
        title={error ? '打开失败，请双击重试' : undefined}
        onDoubleClick={() => void open()}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            void open()
          }
        }}
      >
        <span
          className="pointer-events-none block size-14 opacity-100 transition-opacity duration-500 ease-out [--floating-background:#ffffff] [--floating-border:#ebe7e4] data-[dimmed=true]:opacity-45 data-[dimmed=true]:[--floating-background:#282828] data-[dimmed=true]:[--floating-border:#282828] [&>svg]:size-full [&_[data-part=background]]:transition-[fill,stroke] [&_[data-part=background]]:duration-500 [&_[data-part=background]]:ease-out"
          data-dimmed={dimmed}
          aria-hidden="true"
          dangerouslySetInnerHTML={{ __html: floatingIcon }}
        />
      </button>
      <button
        ref={closeRef}
        type="button"
        className={cn(
          'invisible absolute top-0 right-0 z-20 flex size-[18px] cursor-pointer items-center justify-center rounded-full border border-border bg-card text-muted-foreground opacity-0 outline-none transition-colors duration-150 will-change-[transform,opacity] hover:bg-[#c83f43] hover:text-white focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring',
          !hovered && 'pointer-events-none',
        )}
        aria-label="关闭选项"
        title="关闭或最小化"
        aria-hidden={!hovered}
        tabIndex={hovered ? 0 : -1}
        onClick={closeApp}
      >
        <X className="pointer-events-none size-3" strokeWidth={1.8} />
      </button>
    </div>
  )
}
