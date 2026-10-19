import { MotionConfig } from 'motion/react'
import { Toaster, ToastProvider } from '@/components/ui/toast'
import { TranslatorWorkspace } from '@/features/translator/TranslatorWorkspace'
import { isDesktop } from '@/lib/desktop'

document.documentElement.dataset.desktop = String(isDesktop)

export function App() {
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
