import { useEffect, useRef, useState } from 'react'
import {
  GetAudioStatus,
  ListAudioSources,
  StartAudioCapture,
  StopAudioCapture,
} from '../../../wailsjs/go/main/AudioService'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { main } from '../../../wailsjs/go/models'
import { Configure } from '../../../wailsjs/go/main/SubtitleService'
import { isDesktop } from '@/lib/desktop'
import { useToastManager } from '@/components/ui/toast'

export type AudioOption = {
  value: string
  label: string
  kind: string
  icon?: string
}
type Status = Pick<
  main.AudioStatus,
  'state' | 'source' | 'session' | 'level' | 'error'
>
const initialStatus: Status = {
  state: 'idle',
  source: { id: '', label: '', kind: '', pid: 0, icon: '' },
  session: 0,
  level: 0,
  error: '',
}
export const defaultAudioOptions: AudioOption[] = [
  { value: 'system', label: '全局监听', kind: 'system' },
]

export function useAudioCapture() {
  const [sources, setSources] = useState(defaultAudioOptions)
  const [status, setStatus] = useState<Status>(initialStatus)
  const [pending, setPending] = useState(false)
  const busy = useRef(false)
  const refreshing = useRef(false)
  const subtitleConfiguration = useRef({
    pending: null as main.SubtitleSettings | null,
    running: false,
    task: Promise.resolve(),
  })
  const toast = useToastManager()
  const notify = (error: unknown) =>
    toast.add({ id: 'audio-error', title: String(error) })

  function configureSubtitles(settings: main.SubtitleSettings) {
    if (!isDesktop) return Promise.resolve()
    const configuration = subtitleConfiguration.current
    configuration.pending = settings
    if (configuration.running) return configuration.task
    configuration.running = true
    configuration.task = (async () => {
      try {
        // Serialize bridge calls and coalesce rapid slider/drag updates.
        while (configuration.pending) {
          const next = configuration.pending
          configuration.pending = null
          try {
            await Configure(next)
          } catch (error) {
            toast.add({ id: 'subtitle-error', title: String(error) })
          }
        }
      } finally {
        configuration.running = false
      }
    })()
    return configuration.task
  }

  async function refreshSources(silent = false) {
    if (!isDesktop || refreshing.current) return
    refreshing.current = true
    try {
      const sources = await ListAudioSources()
      setSources(
        sources.map((source) => ({
          value: source.id,
          label: source.label,
          kind: source.kind,
          icon: source.icon,
        })),
      )
    } catch (error) {
      if (!silent) notify(error)
    } finally {
      refreshing.current = false
    }
  }

  useEffect(() => {
    if (!isDesktop) return
    let alive = true
    let receivedStatus = false
    const offStatus = EventsOn('audio:status', (next: Status) => {
      receivedStatus = true
      setStatus(next)
      if (next.error) notify(next.error)
    })
    const offSubtitleError = EventsOn('subtitle:error', (error: string) => {
      toast.add({ id: 'subtitle-error', title: error })
    })
    const offChunk = EventsOn(
      'audio:chunk',
      (chunk: { session: number; level: number }) => {
        setStatus((previous) =>
          previous.state === 'listening' && previous.session === chunk.session
            ? { ...previous, level: chunk.level }
            : previous,
        )
      },
    )
    void GetAudioStatus()
      .then((next) => {
        if (alive && !receivedStatus) setStatus(next)
      })
      .catch(notify)
    void refreshSources()
    const refreshVisible = () => {
      if (document.visibilityState === 'visible') void refreshSources(true)
    }
    const timer = window.setInterval(refreshVisible, 2000)
    window.addEventListener('focus', refreshVisible)
    document.addEventListener('visibilitychange', refreshVisible)
    return () => {
      alive = false
      offStatus()
      offSubtitleError()
      offChunk()
      window.clearInterval(timer)
      window.removeEventListener('focus', refreshVisible)
      document.removeEventListener('visibilitychange', refreshVisible)
    }
  }, [])

  async function command(source?: string) {
    if (busy.current) return false
    if (!isDesktop) {
      notify('请在桌面应用中启动音频监听')
      return false
    }
    busy.current = true
    setPending(true)
    try {
      if (source) await subtitleConfiguration.current.task
      const next = source
        ? await StartAudioCapture(source)
        : await StopAudioCapture()
      setStatus(next)
      return true
    } catch (error) {
      notify(error)
      return false
    } finally {
      busy.current = false
      setPending(false)
    }
  }

  return {
    sources,
    status,
    pending,
    refreshSources,
    configureSubtitles,
    start: (source: string) => command(source),
    stop: () => command(),
  }
}

export type AudioCapture = ReturnType<typeof useAudioCapture>
