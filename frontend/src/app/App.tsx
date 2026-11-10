import { MotionConfig } from 'motion/react'
import { useEffect, useState } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ScreenshotEditor } from '@/features/screenshot/ScreenshotEditor'
import type { Frame } from '@/features/screenshot/model'
import { Toaster, ToastProvider } from '@/components/ui/toast'
import { TranslatorWorkspace } from '@/features/translator/TranslatorWorkspace'
import { isDesktop } from '@/lib/desktop'
import { FloatingBall } from './FloatingBall'
import { IsWorkspace, ShowWorkspace } from '../../wailsjs/go/app/WindowService'

document.documentElement.dataset.desktop = String(isDesktop)

export function App() {
  const [floating, setFloating] = useState(isDesktop)
  const [capture, setCapture] = useState<Frame | null>(null)
  const [captureError, setCaptureError] = useState('')
  useEffect(() => {
    if (!isDesktop) return
    let cancelled = false
    void IsWorkspace()
      .then((workspace) => {
        if (cancelled) return
        setFloating(!workspace)
      })
      .catch(console.error)
    return () => {
      cancelled = true
    }
  }, [])
  useEffect(() => {
    // Hidden WebViews can suspend animation frames; reveal after the DOM commit.
    if (isDesktop && !floating) void ShowWorkspace().catch(console.error)
  }, [floating])
  useEffect(() => {
    if (!isDesktop) return
    const start = EventsOn('screenshot:start', (frame: Frame) =>
      setCapture(frame),
    )
    const end = EventsOn('screenshot:end', () => setCapture(null))
    const error = EventsOn('screenshot:error', (message: string) =>
      setCaptureError(message),
    )
    return () => {
      start()
      end()
      error()
    }
  }, [])

  return (
    <>
      <div style={{ display: capture ? 'none' : undefined }}>
        {floating ? (
          <FloatingBall />
        ) : (
          <MotionConfig reducedMotion="user">
            <ToastProvider timeout={2800} limit={1}>
              <main className="min-h-svh bg-background">
                <TranslatorWorkspace />
              </main>
              <Toaster />
            </ToastProvider>
          </MotionConfig>
        )}
      </div>
      {capture && <ScreenshotEditor initial={capture} />}
      {captureError && (
        <button
          className="fixed bottom-2 left-2 rounded bg-red-950 p-2 text-xs text-white"
          onClick={() => setCaptureError('')}
        >
          {captureError} · 关闭
        </button>
      )}
    </>
  )
}
