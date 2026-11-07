import { MotionConfig } from 'motion/react'
import { useState } from 'react'
import { Toaster, ToastProvider } from '@/components/ui/toast'
import { TranslatorWorkspace } from '@/features/translator/TranslatorWorkspace'
import { isDesktop } from '@/lib/desktop'
import { FloatingBall } from './FloatingBall'

document.documentElement.dataset.desktop = String(isDesktop)

export function App() {
  const [floating, setFloating] = useState(isDesktop)

  if (floating) return <FloatingBall onOpen={() => setFloating(false)} />

  return (
    <MotionConfig reducedMotion="user">
      <ToastProvider timeout={2800} limit={1}>
        <main className="min-h-svh bg-background">
          <TranslatorWorkspace />
        </main>
        <Toaster />
      </ToastProvider>
    </MotionConfig>
  )
}
