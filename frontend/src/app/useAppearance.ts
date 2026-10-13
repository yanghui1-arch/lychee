import { useLayoutEffect, useState } from 'react'

export type Appearance = 'light' | 'system' | 'dark'
const STORAGE_KEY = 'lychee.appearance.v1'

export function isAppearance(value: unknown): value is Appearance {
  return value === 'light' || value === 'system' || value === 'dark'
}

function readAppearance(): Appearance {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    return isAppearance(saved) ? saved : 'system'
  } catch {
    return 'system'
  }
}

export function useAppearance() {
  const [appearance, setAppearance] = useState(readAppearance)

  useLayoutEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => {
      document.documentElement.classList.toggle(
        'dark',
        appearance === 'dark' || (appearance === 'system' && media.matches),
      )
    }
    apply()
    media.addEventListener('change', apply)
    try {
      localStorage.setItem(STORAGE_KEY, appearance)
    } catch {
      /* Keep appearance switching usable when storage is unavailable. */
    }
    return () => media.removeEventListener('change', apply)
  }, [appearance])

  return { appearance, setAppearance }
}
