import { cn } from '@/lib/utils'
import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { AppWindow, ArrowRightLeft, Monitor } from 'lucide-react'
import { motion } from 'motion/react'
import {
  PopupSelect,
  SegmentedControl,
  SettingsGroup,
  SettingsRow,
} from '@/components/ui/settings'
import { LANGUAGES, type Language } from '@/features/translator/model'
import { ListeningSourceLabel } from './ListeningSourceLabel'
import type { AudioCapture, AudioOption } from './useAudioCapture'
import { useDesktopBackground } from './useDesktopBackground'
import {
  readSubtitleSettings,
  subtitleSamples,
  subtitleStorageKey,
  subtitleStyles,
  type SubtitlePreferences,
} from './model'
import './subtitle.css'

// Inset between the preview edge and the caption, in CSS pixels.
const PREVIEW_INSET = 14
const POSITIONS = [
  { value: 'top', label: '顶部' },
  { value: 'bottom', label: '底部' },
  { value: 'custom', label: '自定义' },
] as const

function AudioSourceIcon({
  source,
  className,
}: {
  source: AudioOption
  className?: string
}) {
  const [failedIcon, setFailedIcon] = useState('')
  if (source.icon && source.icon !== failedIcon) {
    return (
      <img
        src={source.icon}
        alt=""
        aria-hidden="true"
        className={cn('size-4 shrink-0 object-contain', className)}
        width={16}
        height={16}
        onError={() => setFailedIcon(source.icon!)}
      />
    )
  }
  const Icon = source.kind === 'system' ? Monitor : AppWindow
  return (
    <Icon
      className={cn(
        'size-4 shrink-0 text-[color:var(--ui-secondary)]',
        className,
      )}
      strokeWidth={1.6}
      aria-hidden="true"
    />
  )
}

export function SubtitleWorkspace({ audio }: { audio: AudioCapture }) {
  const [settings, setSettings] = useState(readSubtitleSettings)
  const { state } = audio.status
  const enabled = state === 'listening' || state === 'starting'
  // The preview always shows fixed sample captions, never the live stream.
  const sourceText = subtitleSamples[settings.source]
  const translatedText = subtitleSamples[settings.target]
  const background = useDesktopBackground()
  const languages = useRef({ source: settings.source, target: settings.target })
  const audioSources: AudioOption[] = audio.sources.some(
    (source) => source.value === settings.audio,
  )
    ? audio.sources
    : [
        ...audio.sources,
        {
          value: settings.audio,
          label: '所选应用当前不可用',
          kind: 'application',
        },
      ]
  const drag = useRef<{
    id: number
    startX: number
    startY: number
    x: number
    y: number
  } | null>(null)
  const patch = (changes: Partial<SubtitlePreferences>) =>
    setSettings((previous) => ({ ...previous, ...changes }))
  const clamp = (value: number) => Math.min(100, Math.max(0, value))

  useEffect(() => {
    try {
      localStorage.setItem(subtitleStorageKey, JSON.stringify(settings))
    } catch {
      // Keep the preview usable when local storage is unavailable.
    }
  }, [settings])
  useEffect(() => {
    const changed =
      languages.current.source !== settings.source ||
      languages.current.target !== settings.target
    languages.current = { source: settings.source, target: settings.target }
    void audio.configureSubtitles(settings).then(() => {
      // A start message fixes the languages for that WS session. Restart the
      // capture after canceling the old stream when a language changes live.
      if (changed && audio.status.state === 'listening')
        void audio.start(settings.audio)
    })
  }, [settings])

  const custom = settings.position === 'custom'
  const hasSound = audio.status.level > 0.002

  return (
    <section
      className="grid min-h-0 grid-rows-[minmax(220px,_1fr)_auto] gap-4 pt-3 pb-1"
      aria-label="字幕设置与预览"
    >
      <div
        className={cn(
          'subtitle-preview relative isolate min-h-0 overflow-hidden rounded-[14px] bg-[var(--preview-color)] transition-[background-color] duration-200',
          'shadow-[0_0_0_1px_#0000000d,_0_1px_2px_#0000000f,_0_10px_28px_-12px_#00000040]',
          "after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:shadow-[inset_0_0_0_1px_#ffffff1f] after:content-['']",
        )}
        role="region"
        aria-label="字幕效果预览"
        data-position={settings.position}
        style={
          {
            '--subtitle-font-size': `${settings.fontSize * 0.46}px`,
            '--preview-color': background?.color ?? 'var(--ui-group)',
          } as CSSProperties
        }
      >
        {background?.image && (
          <motion.img
            key={background.image}
            src={background.image}
            alt="桌面壁纸"
            draggable={false}
            className="pointer-events-none absolute inset-0 size-full object-cover select-none"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.24 }}
          />
        )}
        {custom && (
          <motion.span
            className="absolute top-3 right-3 flex h-[22px] items-center rounded-full bg-black/20 px-2.5 text-[11px] text-white/90 shadow-[inset_0_0_0_0.5px_#ffffff2e] backdrop-blur-md"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.18 }}
            aria-hidden="true"
          >
            拖动字幕调整位置
          </motion.span>
        )}
        <div
          className="subtitle-caption subtitle-palette focus-visible:outline-solid focus-visible:outline-white"
          data-style={settings.style}
          role={custom ? 'button' : undefined}
          tabIndex={custom ? 0 : undefined}
          aria-label={
            custom ? '自定义字幕位置：拖动或使用方向键调整' : undefined
          }
          title={custom ? '拖动字幕调整位置，方向键微调' : undefined}
          style={
            custom
              ? {
                  left: `calc(${PREVIEW_INSET}px + (100% - ${PREVIEW_INSET * 2}px) * ${settings.customX / 100})`,
                  top: `calc(${PREVIEW_INSET}px + (100% - ${PREVIEW_INSET * 2}px) * ${settings.customY / 100})`,
                  transform: `translate(-${settings.customX}%, -${settings.customY}%)`,
                }
              : undefined
          }
          onPointerDown={(event) => {
            if (!custom || event.button !== 0) return
            event.currentTarget.focus()
            event.currentTarget.setPointerCapture(event.pointerId)
            drag.current = {
              id: event.pointerId,
              startX: event.clientX,
              startY: event.clientY,
              x: settings.customX,
              y: settings.customY,
            }
          }}
          onPointerMove={(event) => {
            const start = drag.current
            if (!start || start.id !== event.pointerId) return
            const caption = event.currentTarget
            const preview = caption.parentElement!
            const span = (outer: number, inner: number) =>
              Math.max(1, outer - inner - PREVIEW_INSET * 2)
            patch({
              customX: clamp(
                start.x +
                  ((event.clientX - start.startX) /
                    span(preview.clientWidth, caption.offsetWidth)) *
                    100,
              ),
              customY: clamp(
                start.y +
                  ((event.clientY - start.startY) /
                    span(preview.clientHeight, caption.offsetHeight)) *
                    100,
              ),
            })
          }}
          onPointerUp={() => {
            drag.current = null
          }}
          onPointerCancel={() => {
            drag.current = null
          }}
          onLostPointerCapture={() => {
            drag.current = null
          }}
          onKeyDown={(event) => {
            const step = {
              ArrowLeft: [-2, 0],
              ArrowRight: [2, 0],
              ArrowUp: [0, -2],
              ArrowDown: [0, 2],
            }[event.key]
            if (!custom || !step) return
            event.preventDefault()
            patch({
              customX: clamp(settings.customX + step[0]),
              customY: clamp(settings.customY + step[1]),
            })
          }}
        >
          <motion.div
            key={`${settings.source}:${settings.target}`}
            initial={{ opacity: 0, y: 3 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, ease: [0.22, 1, 0.36, 1] }}
          >
            {sourceText && (
              <p className="subtitle-ink" lang={settings.source}>
                {sourceText}
              </p>
            )}
            {translatedText && (
              <p className="subtitle-ink" lang={settings.target}>
                {translatedText}
              </p>
            )}
          </motion.div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 [@media(max-width:760px)]:grid-cols-1">
        <SettingsGroup label="字幕">
          <SettingsRow>
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span
                id="subtitle-heading"
                className="text-[13px] font-medium text-[color:var(--ui-label)]"
              >
                桌面字幕
              </span>
              <span
                id="subtitle-description"
                className="flex min-w-0 items-center gap-1.5 text-[11.5px] leading-4 text-[color:var(--ui-secondary)]"
                role="status"
              >
                {state === 'listening' ? (
                  <>
                    <span
                      className="size-1.5 shrink-0 rounded-full bg-[var(--ui-live)] transition-[box-shadow] duration-100"
                      style={{
                        boxShadow: `0 0 0 ${Math.min(4, audio.status.level * 40)}px color-mix(in srgb, var(--ui-live) 28%, transparent)`,
                      }}
                      aria-label={`音量 ${Math.round(audio.status.level * 100)}%`}
                    />
                    <ListeningSourceLabel
                      key={audio.status.session}
                      label={audio.status.source.label}
                      hasSound={hasSound}
                    />
                  </>
                ) : (
                  <span
                    className={cn(
                      'truncate',
                      audio.status.error &&
                        'text-[#d93f3f] dark:text-[#ff7a7a]',
                    )}
                    title={audio.status.error || undefined}
                  >
                    {state === 'starting'
                      ? '正在启动…'
                      : audio.status.error || '实时转写并翻译电脑中的声音'}
                  </span>
                )}
              </span>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={enabled}
              aria-labelledby="subtitle-heading"
              aria-describedby="subtitle-description"
              className={cn(
                'relative inline-flex h-[24px] w-[42px] shrink-0 cursor-pointer items-center rounded-full p-[2px]',
                'bg-[var(--ui-switch-off)] shadow-[var(--ui-track-inset)] transition-[background-color] duration-200',
                'aria-checked:bg-[var(--ui-live)] disabled:cursor-wait disabled:opacity-60',
                'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-ring)]',
              )}
              disabled={audio.pending}
              onClick={() =>
                void (enabled ? audio.stop() : audio.start(settings.audio))
              }
            >
              <span
                className={cn(
                  'size-5 rounded-full bg-white shadow-[0_1px_3px_#00000040,_0_0_0_0.5px_#0000000f]',
                  'transition-transform duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]',
                  enabled && 'translate-x-[18px]',
                )}
              />
            </button>
          </SettingsRow>
          <SettingsRow label="语言">
            <PopupSelect
              label="原语言"
              value={settings.source}
              items={LANGUAGES}
              disabled={audio.pending}
              onChange={(source) => patch({ source: source as Language })}
            />
            <button
              type="button"
              className="grid size-6 shrink-0 cursor-pointer place-items-center rounded-[6px] text-[color:var(--ui-secondary)] transition-colors hover:bg-[var(--ui-track)] hover:text-[color:var(--ui-label)] disabled:cursor-not-allowed disabled:opacity-40 focus-visible:outline-2 focus-visible:outline-[var(--color-ring)]"
              aria-label="交换原语言与目标语言"
              title="交换语言"
              disabled={audio.pending}
              onClick={() =>
                patch({ source: settings.target, target: settings.source })
              }
            >
              <ArrowRightLeft className="size-3.5" strokeWidth={1.75} />
            </button>
            <PopupSelect
              label="目标语言"
              value={settings.target}
              items={LANGUAGES}
              disabled={audio.pending}
              onChange={(target) => patch({ target: target as Language })}
            />
          </SettingsRow>
          <SettingsRow label="来源">
            <PopupSelect
              label="监听来源"
              value={settings.audio}
              items={audioSources}
              disabled={audio.pending}
              renderIcon={(source) => <AudioSourceIcon source={source} />}
              onOpenChange={(open) => {
                if (open) void audio.refreshSources()
              }}
              onChange={(next) => {
                if (enabled) {
                  void audio.start(next).then((success) => {
                    if (success) patch({ audio: next })
                  })
                } else patch({ audio: next })
              }}
            />
          </SettingsRow>
        </SettingsGroup>

        <SettingsGroup label="字幕外观">
          <SettingsRow label="字号" htmlFor="subtitle-font-size">
            <span
              className="text-[11px] text-[color:var(--ui-secondary)]"
              aria-hidden="true"
            >
              A
            </span>
            <input
              id="subtitle-font-size"
              className="subtitle-slider"
              type="range"
              min={18}
              max={40}
              step={1}
              value={settings.fontSize}
              title={`${settings.fontSize}`}
              style={
                {
                  '--slider-progress': `${((settings.fontSize - 18) / 22) * 100}%`,
                } as CSSProperties
              }
              onChange={(event) =>
                patch({ fontSize: Number(event.target.value) })
              }
            />
            <span
              className="text-[16px] text-[color:var(--ui-secondary)]"
              aria-hidden="true"
            >
              A
            </span>
          </SettingsRow>
          <SettingsRow label="位置">
            <SegmentedControl
              name="subtitle-position"
              label="字幕位置"
              value={settings.position}
              options={POSITIONS}
              onChange={(position) => patch({ position })}
            />
          </SettingsRow>
          <SettingsRow label="样式">
            <div
              className="flex flex-1 gap-2"
              role="radiogroup"
              aria-label="字幕样式"
            >
              {subtitleStyles.map((style) => {
                const checked = settings.style === style.value
                return (
                  <label
                    key={style.value}
                    className={cn(
                      'subtitle-palette relative grid h-[26px] min-w-0 flex-1 cursor-pointer place-items-center rounded-[6px] bg-cover bg-[center_62%]',
                      'shadow-[inset_0_0_0_0.5px_#00000026] transition-shadow duration-150',
                      checked &&
                        'shadow-[0_0_0_2px_var(--ui-group),_0_0_0_3.5px_var(--ui-label)]',
                      'has-focus-visible:outline-2 has-focus-visible:outline-offset-4 has-focus-visible:outline-[var(--color-ring)]',
                    )}
                    style={{
                      backgroundColor: background?.color ?? 'var(--ui-track)',
                      backgroundImage: background?.image
                        ? `url(${background.image})`
                        : undefined,
                    }}
                    data-style={style.value}
                    title={style.label}
                  >
                    <input
                      className="sr-only"
                      type="radio"
                      name="subtitle-style"
                      value={style.value}
                      aria-label={style.label}
                      checked={checked}
                      onChange={() => patch({ style: style.value })}
                    />
                    <span className="subtitle-ink rounded-[2px] px-1 text-[12px] leading-[16px]">
                      Aa
                    </span>
                  </label>
                )
              })}
            </div>
          </SettingsRow>
        </SettingsGroup>
      </div>
    </section>
  )
}
