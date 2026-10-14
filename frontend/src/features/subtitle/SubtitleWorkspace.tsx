import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { Mic, Monitor } from 'lucide-react'
import previewBackground from '@/assets/subtitle-preview.webp'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LanguageSelect } from '@/features/translator/LanguageSelect'
import { LANGUAGES, type Language } from '@/features/translator/model'
import './subtitle.css'

const samples: Record<Language, string> = {
  'zh-CN': '未来属于那些不断探索的人。',
  'zh-TW': '未來屬於那些不斷探索的人。',
  'en-US': 'The future belongs to those who keep exploring.',
  'en-GB': 'The future belongs to those who keep exploring.',
  'ja-JP': '未来は、探求し続ける人のものです。',
  'ko-KR': '미래는 끊임없이 탐구하는 사람들의 것입니다.',
  'fr-FR': 'L’avenir appartient à ceux qui continuent d’explorer.',
  'de-DE': 'Die Zukunft gehört denen, die immer weiter entdecken.',
}
const styles = [
  { value: 'charcoal', label: '深色磨砂' },
  { value: 'mist', label: '浅灰字幕' },
  { value: 'paper', label: '白色字幕' },
  { value: 'slate', label: '灰色字幕' },
  { value: 'graphite', label: '深灰字幕' },
] as const
const audioSources = [
  { value: 'system', label: '系统声音' },
  { value: 'microphone', label: '麦克风' },
] as const
type Settings = {
  source: Language
  target: Language
  audio: (typeof audioSources)[number]['value']
  fontSize: number
  position: 'top' | 'bottom' | 'custom'
  customX: number
  customY: number
  style: (typeof styles)[number]['value']
}
const defaults: Settings = {
  source: 'en-US',
  target: 'zh-CN',
  audio: 'system',
  fontSize: 30,
  position: 'bottom',
  customX: 50,
  customY: 82,
  style: 'charcoal',
}
const storageKey = 'lychee.subtitle.v1'

function readSettings(): Settings {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey) || 'null')
    if (!saved || typeof saved !== 'object') return defaults
    return Object.fromEntries(
      Object.entries(defaults).map(([key, fallback]) => {
        const value = saved[key]
        const valid =
          key === 'source' || key === 'target'
            ? LANGUAGES.some((language) => language.value === value)
            : key === 'fontSize'
              ? Number.isInteger(value) && value >= 18 && value <= 40
              : key === 'style'
                ? styles.some((style) => style.value === value)
                : key === 'audio'
                  ? audioSources.some((audio) => audio.value === value)
                  : key === 'position'
                    ? ['top', 'bottom', 'custom'].includes(value)
                    : typeof value === 'number' &&
                      Number.isFinite(value) &&
                      value >= 0 &&
                      value <= 100
        return [key, valid ? value : fallback]
      }),
    ) as Settings
  } catch {
    return defaults
  }
}

export function SubtitleWorkspace() {
  const [settings, setSettings] = useState(readSettings)
  const [enabled, setEnabled] = useState(false)
  const drag = useRef<{
    id: number
    startX: number
    startY: number
    x: number
    y: number
  } | null>(null)
  const patch = (changes: Partial<Settings>) =>
    setSettings((previous) => ({ ...previous, ...changes }))
  const clamp = (value: number) => Math.min(100, Math.max(0, value))
  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(settings))
    } catch {
      // Keep the preview usable when local storage is unavailable.
    }
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
          <p lang={settings.source}>{samples[settings.source]}</p>
          <p lang={settings.target}>{samples[settings.target]}</p>
        </div>
      </div>

      <section className="subtitle-setup" aria-labelledby="subtitle-heading">
        <div className="subtitle-heading-row">
          <div>
            <h1 id="subtitle-heading">桌面字幕</h1>
            <p id="subtitle-description">将电脑中的声音实时转换为桌面字幕</p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            aria-label="桌面字幕"
            aria-describedby="subtitle-description"
            className="subtitle-switch"
            onClick={() => setEnabled((previous) => !previous)}
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
              onValueChange={(audio) => {
                if (audio) patch({ audio })
              }}
            >
              <label className="language-label" htmlFor="subtitle-audio-source">
                监听来源
              </label>
              <SelectTrigger
                id="subtitle-audio-source"
                className="subtitle-audio-trigger"
                aria-label="监听来源"
              >
                {settings.audio === 'system' ? (
                  <Monitor strokeWidth={1.6} />
                ) : (
                  <Mic strokeWidth={1.6} />
                )}
                <SelectValue />
              </SelectTrigger>
              <SelectContent
                className="select-popup"
                align="start"
                sideOffset={8}
                alignItemWithTrigger={false}
              >
                {audioSources.map((audio) => (
                  <SelectItem key={audio.value} value={audio.value}>
                    {audio.value === 'system' ? (
                      <Monitor
                        className="size-5 text-muted-foreground"
                        strokeWidth={1.6}
                      />
                    ) : (
                      <Mic
                        className="size-5 text-muted-foreground"
                        strokeWidth={1.6}
                      />
                    )}
                    {audio.label}
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
            {styles.map((style) => (
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
