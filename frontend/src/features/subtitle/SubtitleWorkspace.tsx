import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { AppWindow, Monitor } from 'lucide-react'
import previewBackground from '@/assets/subtitle-preview.webp'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LanguageSelect } from '@/features/translator/LanguageSelect'
import './subtitle.css'
import { ListeningSourceLabel } from './ListeningSourceLabel'
import type { AudioCapture, AudioOption } from './useAudioCapture'
import {
  readSubtitleSettings,
  subtitleSamples,
  subtitleStorageKey,
  subtitleStyles,
  type SubtitlePreferences,
} from './model'

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
        className={`subtitle-app-icon ${className ?? ''}`}
        width={20}
        height={20}
        onError={() => setFailedIcon(source.icon!)}
      />
    )
  }
  const Icon = source.kind === 'system' ? Monitor : AppWindow
  return <Icon className={className} strokeWidth={1.6} />
}

export function SubtitleWorkspace({ audio }: { audio: AudioCapture }) {
  const [settings, setSettings] = useState(readSubtitleSettings)
  const enabled =
    audio.status.state === 'listening' || audio.status.state === 'starting'
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
  const selectedAudio = audioSources.find(
    (source) => source.value === settings.audio,
  )!
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
    void audio.configureSubtitles(settings)
  }, [settings])

  return (
    <section className="subtitle-workspace" aria-label="字幕设置与预览">
      <div
        className="subtitle-preview"
        role="region"
        aria-label="字幕效果预览"
        data-position={settings.position}
        style={
          {
            '--subtitle-font-size': `${settings.fontSize * 0.46}px`,
          } as CSSProperties
        }
      >
        <img
          src={previewBackground}
          alt="夕阳下的群山与云海"
          draggable={false}
        />
        <span className="subtitle-preview-label">预览</span>
        <div
          className="subtitle-caption"
          data-style={settings.style}
          role={settings.position === 'custom' ? 'button' : undefined}
          tabIndex={settings.position === 'custom' ? 0 : undefined}
          aria-label={
            settings.position === 'custom'
              ? '自定义字幕位置：拖动或使用方向键调整'
              : undefined
          }
          title={
            settings.position === 'custom'
              ? '拖动字幕调整位置，方向键微调'
              : undefined
          }
          style={
            settings.position === 'custom'
              ? {
                  left: `calc(12px + (100% - 24px) * ${settings.customX / 100})`,
                  top: `calc(12px + (100% - 24px) * ${settings.customY / 100})`,
                  transform: `translate(-${settings.customX}%, -${settings.customY}%)`,
                }
              : undefined
          }
          onPointerDown={(event) => {
            if (settings.position !== 'custom' || event.button !== 0) return
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
            patch({
              customX: clamp(
                start.x +
                  ((event.clientX - start.startX) /
                    Math.max(
                      1,
                      preview.clientWidth - caption.offsetWidth - 24,
                    )) *
                    100,
              ),
              customY: clamp(
                start.y +
                  ((event.clientY - start.startY) /
                    Math.max(
                      1,
                      preview.clientHeight - caption.offsetHeight - 24,
                    )) *
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
            if (
              settings.position !== 'custom' ||
              !['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(
                event.key,
              )
            )
              return
            event.preventDefault()
            patch({
              customX: clamp(
                settings.customX +
                  (event.key === 'ArrowLeft'
                    ? -2
                    : event.key === 'ArrowRight'
                      ? 2
                      : 0),
              ),
              customY: clamp(
                settings.customY +
                  (event.key === 'ArrowUp'
                    ? -2
                    : event.key === 'ArrowDown'
                      ? 2
                      : 0),
              ),
            })
          }}
        >
          <p lang={settings.source}>{subtitleSamples[settings.source]}</p>
          <p lang={settings.target}>{subtitleSamples[settings.target]}</p>
        </div>
      </div>

      <section className="subtitle-setup" aria-labelledby="subtitle-heading">
        <div className="subtitle-heading-row">
          <h1 id="subtitle-heading">桌面字幕</h1>
          <p
            id="subtitle-description"
            className="subtitle-audio-status"
            role="status"
          >
            {audio.status.state === 'listening' ? (
              <>
                <span
                  className="subtitle-audio-meter"
                  aria-label={`音量 ${Math.round(audio.status.level * 100)}%`}
                >
                  <span
                    style={{
                      width: `${Math.min(100, audio.status.level * 500)}%`,
                    }}
                  />
                </span>
                <ListeningSourceLabel
                  key={audio.status.session}
                  label={audio.status.source.label}
                  hasSound={audio.status.level > 0.002}
                />
              </>
            ) : (
              <span className="subtitle-audio-description">
                {audio.status.state === 'starting'
                  ? '正在连接音频来源…'
                  : audio.status.error || '将电脑中的声音实时转换为桌面字幕'}
              </span>
            )}
          </p>
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            aria-label="桌面字幕"
            aria-describedby="subtitle-description"
            className="subtitle-switch"
            disabled={audio.pending}
            onClick={() =>
              void (enabled ? audio.stop() : audio.start(settings.audio))
            }
          >
            <span />
          </button>
        </div>
        <div className="subtitle-source-row">
          <div className="subtitle-language">
            <LanguageSelect
              label="原语言"
              value={settings.source}
              onChange={(source) => patch({ source })}
            />
          </div>
          <div className="subtitle-language">
            <LanguageSelect
              label="目标语言"
              value={settings.target}
              onChange={(target) => patch({ target })}
            />
          </div>
          <div className="subtitle-audio">
            <Select
              items={audioSources}
              value={settings.audio}
              disabled={audio.pending}
              onOpenChange={(open) => {
                if (open) void audio.refreshSources()
              }}
              onValueChange={(next) => {
                if (!next) return
                if (enabled) {
                  void audio.start(next).then((success) => {
                    if (success) patch({ audio: next })
                  })
                } else patch({ audio: next })
              }}
            >
              <label className="language-label" htmlFor="subtitle-audio-source">
                监听来源
              </label>
              <SelectTrigger
                id="subtitle-audio-source"
                className="subtitle-audio-trigger"
                aria-label="监听来源"
                title={selectedAudio.label}
              >
                <AudioSourceIcon source={selectedAudio} />
                <SelectValue />
              </SelectTrigger>
              <SelectContent
                className="select-popup subtitle-audio-popup"
                align="start"
                sideOffset={8}
                alignItemWithTrigger={false}
              >
                {audioSources.map((source) => (
                  <SelectItem
                    key={source.value}
                    value={source.value}
                    className="subtitle-audio-option"
                    title={source.label}
                  >
                    <AudioSourceIcon
                      source={source}
                      className="size-5 text-muted-foreground"
                    />
                    <span className="subtitle-audio-label">{source.label}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </section>

      <section className="subtitle-controls" aria-label="字幕外观">
        <div className="subtitle-control subtitle-font-control">
          <label
            className="subtitle-control-label"
            htmlFor="subtitle-font-size"
          >
            字号
          </label>
          <div className="subtitle-slider-row">
            <input
              id="subtitle-font-size"
              type="range"
              min={18}
              max={40}
              step={1}
              value={settings.fontSize}
              style={
                {
                  '--slider-progress': `${((settings.fontSize - 18) / 22) * 100}%`,
                } as CSSProperties
              }
              onChange={(event) =>
                patch({ fontSize: Number(event.target.value) })
              }
            />
            <output htmlFor="subtitle-font-size">{settings.fontSize}</output>
          </div>
        </div>
        <div
          className="subtitle-control subtitle-position-control"
          role="group"
          aria-labelledby="subtitle-position-label"
        >
          <span id="subtitle-position-label" className="subtitle-control-label">
            位置
          </span>
          <div
            className="subtitle-position-options"
            role="radiogroup"
            aria-label="字幕位置"
          >
            {(['top', 'bottom', 'custom'] as const).map((position) => (
              <label key={position} className="subtitle-position-option">
                <input
                  type="radio"
                  name="subtitle-position"
                  value={position}
                  checked={settings.position === position}
                  onChange={() => patch({ position })}
                />
                <span>
                  {position === 'top'
                    ? '顶部'
                    : position === 'bottom'
                      ? '底部'
                      : '自定义'}
                </span>
              </label>
            ))}
          </div>
        </div>
        <div
          className="subtitle-control subtitle-style-control"
          role="group"
          aria-labelledby="subtitle-style-label"
        >
          <span id="subtitle-style-label" className="subtitle-control-label">
            字幕样式
          </span>
          <div
            className="subtitle-style-options"
            role="radiogroup"
            aria-label="字幕样式"
          >
            {subtitleStyles.map((style) => (
              <label
                key={style.value}
                className="subtitle-style-option"
                data-style={style.value}
                title={style.label}
              >
                <input
                  type="radio"
                  name="subtitle-style"
                  value={style.value}
                  aria-label={style.label}
                  checked={settings.style === style.value}
                  onChange={() => patch({ style: style.value })}
                />
                <span>Aa</span>
              </label>
            ))}
          </div>
        </div>
      </section>
    </section>
  )
}
