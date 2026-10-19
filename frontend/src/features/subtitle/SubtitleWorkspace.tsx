import { cn } from '@/lib/utils'
import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { AppWindow, Monitor } from 'lucide-react'
import { motion } from 'motion/react'
import previewBackground from '@/assets/subtitle-preview.webp'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LanguageSelect } from '@/features/translator/LanguageSelect'
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
        className={`shrink-0 w-[20px] h-[20px] object-contain ${className ?? ''}`}
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
  const live = enabled || !!audio.frame.key
  const sourceText = live
    ? audio.frame.source
    : subtitleSamples[settings.source]
  const translatedText = live
    ? audio.frame.translation
    : subtitleSamples[settings.target]
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

  return (
    <section
      className="grid grid-rows-[minmax(200px,_1fr)_auto_auto] gap-7 min-h-0 pt-4.5 px-[21px] pb-11.5 [@media(max-width:760px)]:grid-rows-[230px_auto_auto] [@media(max-width:760px)]:pt-4.5 [@media(max-width:760px)]:px-3 [@media(max-width:760px)]:pb-7.5"
      aria-label="字幕设置与预览"
    >
      <div
        className={cn(
          'relative min-h-0 overflow-hidden rounded-[9px] bg-[#b0c4d6] isolate [&_>_img]:absolute [&_>_img]:w-full [&_>_img]:h-full',
          '[&_>_img]:object-cover [&_>_img]:select-none [&_>_img]:pointer-events-none',
          "[&[data-position='top']_.subtitle-caption]:[top:12px] [&[data-position='top']_.subtitle-caption]:[bottom:auto]",
          "[&[data-position='custom']_.subtitle-caption]:[bottom:auto] [&[data-position='custom']_.subtitle-caption]:cursor-grab",
          "[&[data-position='custom']_.subtitle-caption]:touch-none [&[data-position='custom']_.subtitle-caption]:select-none",
          "[&[data-position='custom']_.subtitle-caption:active]:cursor-grabbing",
        )}
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
        <span className="absolute [top:14px] [left:17px] py-[3px] px-[9px] rounded-[99px] bg-[#45617e42] text-white text-[10px] leading-[1.4]">
          {live
            ? !audio.frame.key
              ? '等待声音'
              : audio.frame.incomplete
                ? '译文未完成'
                : '字幕阅读中'
            : '预览'}
          {live &&
            audio.frame.pendingSeconds > 5 &&
            ` · 待读约 ${audio.frame.pendingSeconds} 秒`}
        </span>
        <div
          className={cn(
            'subtitle-caption absolute [bottom:12px] [left:50%] [transform:translateX(-50%)] box-border w-[min(460px,_calc(100%_-_24px))]',
            'h-[110.4px] py-[8.28px] px-[16.56px] overflow-hidden border-0 rounded-[0] text-[color:var(--caption-foreground)]',
            'text-[length:var(--subtitle-font-size)] leading-[1.2] text-center [transition:background_180ms_ease,_color_180ms_ease]',
            '[&_>_div]:grid [&_>_div]:h-full [&_>_div]:grid-rows-[repeat(2,_minmax(0,_1fr))] [&_>_div]:gap-[5.52px]',
            "[&_>_div[data-bilingual='false']]:grid-rows-[minmax(0,_1fr)] [&_p]:[align-self:end] [&_p]:min-w-0 [&_p]:justify-self-center",
            '[&_p]:w-fit [&_p]:max-w-[calc(100%_+_0.72em)] [&_p]:max-h-[1.52em] [&_p]:overflow-hidden [&_p]:box-border [&_p]:my-[-0.16em]',
            '[&_p]:mx-[-0.36em] [&_p]:py-[0.16em] [&_p]:px-[0.36em] [&_p]:bg-[var(--caption-background)]',
            "[&_p]:[transition:background_180ms_ease] [&_p]:whitespace-nowrap [&_p]:font-[family-name:'Lychee_Roboto',_sans-serif]",
            '[&_p]:font-medium [&_p]:[-webkit-text-stroke:var(--caption-stroke)_var(--caption-outline)] [&_p]:[paint-order:stroke_fill]',
            '[&_p]:[text-shadow:0_0.5px_1px_var(--caption-shadow)] [&_p_>_span]:block [&_p_>_span]:max-h-[1.2em]',
            '[&_p_>_span]:overflow-hidden',
            "[&[data-empty='true']]:invisible [&_p:lang(zh)]:font-[family-name:'Microsoft_YaHei_UI',_'Segoe_UI',_sans-serif]",
            "[&_p:lang(zh)]:font-normal [&_p:lang(ja)]:font-[family-name:'Yu_Gothic_UI',_'Segoe_UI',_sans-serif] [&_p:lang(ja)]:font-normal",
            "[&_p:lang(ko)]:font-[family-name:'Malgun_Gothic',_'Segoe_UI',_sans-serif] [&_p:lang(ko)]:font-normal",
            "[&_p_+_p]:[align-self:start] [&_p_+_p]:text-[length:0.95em] [&_>_div[data-bilingual='false']_p]:self-center",
            'focus-visible:[outline:2px_solid_#fff] focus-visible:[outline-offset:3px] [--caption-stroke:0.55px]',
            '[--caption-background:transparent] [--caption-foreground:#fff] [--caption-outline:#000000dc] [--caption-shadow:#00000079]',
            "[&[data-style='charcoal']]:[--caption-stroke:0.25px] [&[data-style='charcoal']]:[--caption-background:#000000b3]",
            "[&[data-style='charcoal']]:[--caption-outline:#00000060] [&[data-style='charcoal']]:[--caption-shadow:#00000035]",
            "[&[data-style='mist']]:[--caption-background:#0f172a1a] [&[data-style='mist']]:[--caption-outline:#000000cc]",
            "[&[data-style='mist']]:[--caption-shadow:#00000070] [&[data-style='paper']]:[--caption-background:#ffffff26]",
            "[&[data-style='paper']]:[--caption-foreground:#20242c] [&[data-style='paper']]:[--caption-outline:#ffffffe6]",
            "[&[data-style='paper']]:[--caption-shadow:#ffffff7f] [&[data-style='slate']]:[--caption-foreground:#ffe9a3]",
            "[&[data-style='slate']]:[--caption-outline:#000000e6] [&[data-style='slate']]:[--caption-shadow:#0000007f]",
            "[&[data-style='graphite']]:[--caption-background:#0f172a33] [&[data-style='graphite']]:[--caption-foreground:#e6edf7]",
          )}
          data-style={settings.style}
          data-empty={!sourceText && !translatedText}
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
          <motion.div
            data-bilingual={!!sourceText && !!translatedText}
            key={live ? audio.frame.key : 'preview'}
            initial={{ opacity: 0, y: 3 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, ease: [0.22, 1, 0.36, 1] }}
          >
            {sourceText && (
              <p lang={settings.source}>
                <span>{sourceText}</span>
              </p>
            )}
            {translatedText && (
              <p lang={settings.target}>
                <span>{translatedText}</span>
              </p>
            )}
          </motion.div>
        </div>
      </div>

      <section
        className="flex flex-col gap-4.5"
        aria-labelledby="subtitle-heading"
      >
        <div
          className={cn(
            'grid grid-cols-[max-content_minmax(0,_1fr)_auto] gap-x-2 gap-y-[3px] [align-items:start] [&_h1]:[grid-column:1]',
            '[&_h1]:[grid-row:1] [&_h1]:m-0 [&_h1]:text-[18px] [&_h1]:font-[650] [&_h1]:leading-[1.4] [&_p]:m-0',
            '[&_p]:text-[color:var(--subtle-foreground)] [&_p]:text-[13px] [&_p]:leading-[1.5]',
          )}
        >
          <h1 id="subtitle-heading">桌面字幕</h1>
          <p
            id="subtitle-description"
            className="grid [grid-column:1_/_3] [grid-row:2] grid-cols-[subgrid] items-center min-w-0"
            role="status"
          >
            {audio.status.state === 'listening' ? (
              <>
                <span
                  className="[grid-column:1] w-full h-[5px] overflow-hidden rounded-[99px] bg-border [&_>_span]:block [&_>_span]:h-full [&_>_span]:bg-[#39a66b] [&_>_span]:[transition:width_100ms_ease]"
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
              <span className="[grid-column:1_/_-1] min-w-0">
                {audio.status.state === 'starting'
                  ? '正在初始化语音监听…'
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
            className={cn(
              '[grid-column:3] [grid-row:1] inline-flex shrink-0 items-center w-[48px] h-[26px] p-[3px] border-0 rounded-[99px]',
              '[background:color-mix(_in_srgb,_var(--subtle-foreground)_40%,_var(--color-card)_)] [transition:background_180ms_ease]',
              'aria-checked:bg-foreground [&_>_span]:w-[20px] [&_>_span]:h-[20px] [&_>_span]:rounded-[50%] [&_>_span]:bg-background',
              '[&_>_span]:shadow-[0_2px_4px_#00000020] [&_>_span]:[transition:transform_180ms_ease]',
              "[&[aria-checked='true']_>_span]:[transform:translateX(22px)] [&:disabled]:cursor-wait [&:disabled]:opacity-[0.6]",
              'cursor-pointer focus-visible:[outline:2px_solid_var(--color-ring)] focus-visible:outline-offset-2',
            )}
            disabled={audio.pending}
            onClick={() =>
              void (enabled ? audio.stop() : audio.start(settings.audio))
            }
          >
            <span />
          </button>
        </div>
        <div className="grid grid-cols-[1fr_0.94fr_1fr] gap-6.5 [&_.language-label]:text-[13px] [@media(max-width:760px)]:grid-cols-[1fr] [@media(max-width:760px)]:gap-3.5">
          <div
            className={cn(
              'flex flex-col gap-1.5 min-w-0 [&_.language-trigger]:w-full [&_.language-trigger]:h-[36px] [&_.language-trigger]:py-0',
              '[&_.language-trigger]:px-[13px] [&_.language-trigger]:border [&_.language-trigger]:border-border',
              '[&_.language-trigger]:rounded-[9px] [&_.language-trigger]:bg-background [&_.language-trigger]:shadow-none',
              '[&_.language-trigger]:text-foreground [&_.language-trigger]:text-[14px] [&_.language-trigger]:font-normal',
            )}
          >
            <LanguageSelect
              label="原语言"
              disabled={audio.pending}
              value={settings.source}
              onChange={(source) => patch({ source })}
            />
          </div>
          <div
            className={cn(
              'flex flex-col gap-1.5 min-w-0 [&_.language-trigger]:w-full [&_.language-trigger]:h-[36px] [&_.language-trigger]:py-0',
              '[&_.language-trigger]:px-[13px] [&_.language-trigger]:border [&_.language-trigger]:border-border',
              '[&_.language-trigger]:rounded-[9px] [&_.language-trigger]:bg-background [&_.language-trigger]:shadow-none',
              '[&_.language-trigger]:text-foreground [&_.language-trigger]:text-[14px] [&_.language-trigger]:font-normal',
            )}
          >
            <LanguageSelect
              label="目标语言"
              disabled={audio.pending}
              value={settings.target}
              onChange={(target) => patch({ target })}
            />
          </div>
          <div className="flex flex-col gap-1.5 min-w-0">
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
              <label
                className="language-label shrink-0 text-[13px] text-[color:var(--subtle-foreground)]"
                htmlFor="subtitle-audio-source"
              >
                监听来源
              </label>
              <SelectTrigger
                id="subtitle-audio-source"
                className={cn(
                  "[&[data-slot='select-trigger']]:w-full [&[data-slot='select-trigger']]:h-[36px] [&[data-slot='select-trigger']]:py-0",
                  "[&[data-slot='select-trigger']]:px-[13px] [&[data-slot='select-trigger']]:border [&[data-slot='select-trigger']]:border-border",
                  "[&[data-slot='select-trigger']]:rounded-[9px] [&[data-slot='select-trigger']]:bg-background",
                  "[&[data-slot='select-trigger']]:shadow-none [&[data-slot='select-trigger']]:text-foreground",
                  "[&[data-slot='select-trigger']]:text-[14px] [&[data-slot='select-trigger']]:font-normal",
                  "[&[data-slot='select-trigger']]:gap-[13px] [&_>_svg:first-child]:w-[20px] [&_>_svg:first-child]:h-[20px]",
                  '[&_>_svg:first-child]:text-[color:var(--subtle-foreground)] [&_>_svg:last-child]:w-[14px] [&_>_svg:last-child]:h-[14px]',
                  "[&_>_svg:last-child]:text-[color:var(--subtle-foreground)] [&_[data-slot='select-value']]:block",
                  "[&_[data-slot='select-value']]:min-w-0 [&_[data-slot='select-value']]:overflow-hidden",
                  "[&_[data-slot='select-value']]:text-ellipsis",
                )}
                aria-label="监听来源"
                title={selectedAudio.label}
              >
                <AudioSourceIcon source={selectedAudio} className="size-5" />
                <SelectValue />
              </SelectTrigger>
              <SelectContent
                className="text-[12px] border-border rounded-[12px] [animation-duration:var(--motion-duration)] [animation-timing-function:var(--motion-ease)] data-ending-style:[animation-duration:var(--motion-close-duration)] w-[min(420px,_calc(100vw_-_24px))] min-w-[var(--anchor-width)]"
                align="start"
                sideOffset={8}
                alignItemWithTrigger={false}
              >
                {audioSources.map((source) => (
                  <SelectItem
                    key={source.value}
                    value={source.value}
                    className="[&_>_:first-child]:min-w-0 [&_>_:first-child]:shrink-1"
                    title={source.label}
                  >
                    <AudioSourceIcon
                      source={source}
                      className="size-5 text-muted-foreground"
                    />
                    <span className="min-w-0 overflow-hidden text-ellipsis">
                      {source.label}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </section>

      <section
        className="grid grid-cols-[1fr_1fr_1.08fr] items-center gap-[25px] mt-2.5 [@media(max-width:760px)]:grid-cols-[1fr] [@media(max-width:760px)]:gap-5.5"
        aria-label="字幕外观"
      >
        <div className="subtitle-font-control flex items-center gap-3 min-w-0">
          <label
            className="shrink-0 text-[13px] text-foreground whitespace-nowrap"
            htmlFor="subtitle-font-size"
          >
            字号
          </label>
          <div
            className={cn(
              'flex flex-1 items-center gap-3.5 min-w-0 [&_input]:w-full [&_input]:min-w-0 [&_input]:h-[4px] [&_input]:m-0',
              '[&_input]:rounded-[99px]',
              '[&_input]:[background:linear-gradient(_to_right,_var(--color-foreground)_var(--slider-progress),_var(--color-border)_var(--slider-progress)_)]',
              '[&_input]:appearance-none [&_input]:cursor-pointer [&_input::-webkit-slider-thumb]:w-[20px]',
              '[&_input::-webkit-slider-thumb]:h-[20px] [&_input::-webkit-slider-thumb]:border [&_input::-webkit-slider-thumb]:border-border',
              '[&_input::-webkit-slider-thumb]:rounded-[50%] [&_input::-webkit-slider-thumb]:bg-card',
              '[&_input::-webkit-slider-thumb]:shadow-[0_2px_5px_#00000020] [&_input::-webkit-slider-thumb]:appearance-none',
              '[&_input:focus-visible]:[outline:2px_solid_var(--color-ring)] [&_input:focus-visible]:[outline-offset:8px]',
              '[&_output]:min-w-[18px] [&_output]:text-[color:var(--subtle-foreground)] [&_output]:text-[13px] [&_output]:tabular-nums',
            )}
          >
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
          className="subtitle-position-control flex items-center gap-3 min-w-0"
          role="group"
          aria-labelledby="subtitle-position-label"
        >
          <span
            id="subtitle-position-label"
            className="shrink-0 text-[13px] text-foreground whitespace-nowrap"
          >
            位置
          </span>
          <div
            className="flex flex-1 h-[30px] border border-border rounded-[99px] bg-muted"
            role="radiogroup"
            aria-label="字幕位置"
          >
            {(['top', 'bottom', 'custom'] as const).map((position) => (
              <label
                key={position}
                className={cn(
                  'relative flex-1 cursor-pointer [&_>_span]:grid [&_>_span]:place-items-center [&_>_span]:h-full [&_>_span]:py-0 [&_>_span]:px-2',
                  '[&_>_span]:rounded-[99px] [&_>_span]:text-[color:var(--subtle-foreground)] [&_>_span]:text-[12px] [&_>_span]:whitespace-nowrap',
                  '[&_>_span]:[transition:background_180ms_ease,_color_180ms_ease] [&_input]:absolute [&_input]:w-[1px] [&_input]:h-[1px]',
                  '[&_input]:opacity-0 [&_input:checked_+_span]:bg-foreground [&_input:checked_+_span]:text-background',
                  '[&_input:focus-visible_+_span]:[outline:2px_solid_var(--color-ring)] [&_input:focus-visible_+_span]:[outline-offset:3px]',
                )}
              >
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
          className="flex items-center min-w-0 gap-2.5"
          role="group"
          aria-labelledby="subtitle-style-label"
        >
          <span
            id="subtitle-style-label"
            className="shrink-0 text-[13px] text-foreground whitespace-nowrap"
          >
            字幕样式
          </span>
          <div
            className="flex flex-1 gap-[7px] h-[30px]"
            role="radiogroup"
            aria-label="字幕样式"
          >
            {subtitleStyles.map((style) => (
              <label
                key={style.value}
                className={cn(
                  '[--caption-stroke:0.55px] [--caption-background:transparent] [--caption-foreground:#fff] [--caption-outline:#000000dc]',
                  "[--caption-shadow:#00000079] [&[data-style='charcoal']]:[--caption-stroke:0.25px]",
                  "[&[data-style='charcoal']]:[--caption-background:#000000b3] [&[data-style='charcoal']]:[--caption-outline:#00000060]",
                  "[&[data-style='charcoal']]:[--caption-shadow:#00000035] [&[data-style='mist']]:[--caption-background:#0f172a1a]",
                  "[&[data-style='mist']]:[--caption-outline:#000000cc] [&[data-style='mist']]:[--caption-shadow:#00000070]",
                  "[&[data-style='paper']]:[--caption-background:#ffffff26] [&[data-style='paper']]:[--caption-foreground:#20242c]",
                  "[&[data-style='paper']]:[--caption-outline:#ffffffe6] [&[data-style='paper']]:[--caption-shadow:#ffffff7f]",
                  "[&[data-style='slate']]:[--caption-foreground:#ffe9a3] [&[data-style='slate']]:[--caption-outline:#000000e6]",
                  "[&[data-style='slate']]:[--caption-shadow:#0000007f] [&[data-style='graphite']]:[--caption-background:#0f172a33]",
                  "[&[data-style='graphite']]:[--caption-foreground:#e6edf7] [&_input]:absolute [&_input]:w-[1px] [&_input]:h-[1px]",
                  '[&_input]:opacity-0 [&_input:focus-visible_+_span]:[outline:2px_solid_var(--color-ring)]',
                  '[&_input:focus-visible_+_span]:[outline-offset:3px] relative flex-1 min-w-0 cursor-pointer rounded-[2px]',
                  '[background:repeating-conic-gradient(#dfe4ea_0%_25%,_#eef1f5_0%_50%)_50%_/_8px_8px] [&_>_span]:grid',
                  '[&_>_span]:place-items-center [&_>_span]:h-full [&_>_span]:border-0 [&_>_span]:rounded-[2px]',
                  '[&_>_span]:bg-[var(--caption-background)] [&_>_span]:text-[color:var(--caption-foreground)] [&_>_span]:text-[12px]',
                  "[&_>_span]:font-[family-name:'Lychee_Roboto',_sans-serif] [&_>_span]:font-medium",
                  '[&_>_span]:[-webkit-text-stroke:0.45px_var(--caption-outline)] [&_>_span]:[paint-order:stroke_fill]',
                  '[&_>_span]:[text-shadow:0_0.5px_1px_var(--caption-shadow)]',
                  '[&_input:checked_+_span]:[outline:1.5px_solid_var(--color-foreground)] [&_input:checked_+_span]:[outline-offset:2px]',
                )}
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
