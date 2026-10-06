import { Toast } from '@base-ui/react/toast'
import { Info } from 'lucide-react'

export const ToastProvider = Toast.Provider
export const useToastManager = Toast.useToastManager

export function Toaster() {
  const { toasts } = useToastManager()
  return (
    <Toast.Portal>
      <Toast.Viewport className="toast-viewport" aria-label="提示通知">
        {toasts.map((toast) => (
          <Toast.Root
            key={toast.id}
            toast={toast}
            swipeDirection={[]}
            className={(state) =>
              `toast t-toast${state.transitionStatus ? '' : ' is-open'}`
            }
          >
            <Toast.Content className="toast-content">
              <Info size={15} aria-hidden="true" />
              <Toast.Title />
            </Toast.Content>
          </Toast.Root>
        ))}
      </Toast.Viewport>
    </Toast.Portal>
  )
}
