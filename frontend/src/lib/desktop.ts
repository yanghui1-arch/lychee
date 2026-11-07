import {
  ClipboardSetText,
  Environment,
  WindowMinimise,
} from '../../wailsjs/runtime/runtime'
import { RequestClose } from '../../wailsjs/go/app/WindowService'

export const isDesktop = 'runtime' in window
export const browserPlatform = /Mac/i.test(navigator.platform)
  ? 'darwin'
  : /Linux/i.test(navigator.platform)
    ? 'linux'
    : 'windows'

export async function getPlatform() {
  return isDesktop ? (await Environment()).platform : browserPlatform
}

export function closeApp() {
  if (isDesktop) void RequestClose().catch(console.error)
}

export function minimizeApp() {
  if (isDesktop) WindowMinimise()
}

export async function copyText(text: string) {
  if (isDesktop) {
    if (!(await ClipboardSetText(text)))
      throw new Error('Clipboard unavailable')
  } else {
    await navigator.clipboard.writeText(text)
  }
}
