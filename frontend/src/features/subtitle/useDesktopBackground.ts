import { useEffect, useState } from 'react'
import fallbackImage from '@/assets/subtitle-preview.webp'
import { isDesktop } from '@/lib/desktop'
import { DesktopBackground } from '../../../wailsjs/go/subtitle/SubtitleService'

export type PreviewBackground = { image: string; color: string }

const fallback: PreviewBackground = { image: fallbackImage, color: '#b0c4d6' }
// Survives tab switches so revisiting the page does not flash a placeholder.
let lastBackground: PreviewBackground | null = isDesktop ? null : fallback

async function readBackground(): Promise<PreviewBackground> {
  try {
    const { image, color } = await DesktopBackground()
    // A solid-color desktop has no image; show that color as-is.
    if (image || color) return { image, color: color || fallback.color }
  } catch (error) {
    console.error(error)
  }
  return fallback
}

/** The user's desktop wallpaper, or null while it is first loading. */
export function useDesktopBackground() {
  const [background, setBackground] = useState(lastBackground)
  useEffect(() => {
    if (!isDesktop) return
    let alive = true
    const refresh = () =>
      void readBackground().then((next) => {
        if (!alive) return
        lastBackground = next
        setBackground((previous) =>
          previous?.image === next.image && previous.color === next.color
            ? previous
            : next,
        )
      })
    refresh()
    // The wallpaper may change while the app is in the background.
    window.addEventListener('focus', refresh)
    return () => {
      alive = false
      window.removeEventListener('focus', refresh)
    }
  }, [])
  return background
}
