import { Toast } from '@base-ui/react/toast'
import { Info } from 'lucide-react'
import { cn } from '@/lib/utils'

// Open/close timing: Transitions.dev, Jakub Antalik.
// License: frontend/licenses/transitions-LICENSE.txt

export const ToastProvider = Toast.Provider
export const useToastManager = Toast.useToastManager

export function Toaster() {
  const { toasts } = useToastManager()
  return (
    <Toast.Portal>
      <Toast.Viewport
        className="pointer-events-none fixed bottom-[25px] left-1/2 z-50 -translate-x-1/2"
        aria-label="提示通知"
      >
        {toasts.map((toast) => (
          <Toast.Root
            key={toast.id}
            toast={toast}
            swipeDirection={[]}
            className={(state) =>
              cn(
                'pointer-events-auto w-max max-w-[calc(100vw-32px)] rounded-[12px] border border-border bg-background px-4 py-2.5 text-[12px] shadow-[0_4px_20px_#0000000a] opacity-0 [transform:translateY(16px)_scale(0.97)] blur-[2px] will-change-[transform,opacity,filter] transition-[opacity,transform,filter] duration-250 ease-[cubic-bezier(0.22,1,0.36,1)] motion-reduce:transition-none!',
                !state.transitionStatus &&
                  'opacity-100 [transform:translateY(0)_scale(1)] blur-none duration-350',
              )
            }
          >
            <Toast.Content className="flex items-center gap-2">
              <Info size={15} aria-hidden="true" />
              <Toast.Title />
            </Toast.Content>
          </Toast.Root>
        ))}
      </Toast.Viewport>
    </Toast.Portal>
  )
}
