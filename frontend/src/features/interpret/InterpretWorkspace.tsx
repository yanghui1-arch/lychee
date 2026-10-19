import { cn } from '@/lib/utils'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import {
  ArrowRight,
  AudioLines,
  Globe,
  Keyboard,
  Mic,
  Volume2,
  Zap,
  type LucideIcon,
} from 'lucide-react'
import listenIcon from '@/assets/interpret-listen.svg'
import speakIcon from '@/assets/interpret-speak.svg'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useToastManager } from '@/components/ui/toast'
import {
  LANGUAGES,
  SOURCE_LANGUAGES,
  type Language,
  type SourceLanguage,
} from '@/features/translator/model'

type Settings = {
  listenSource: SourceLanguage
  listenTarget: Language
  listenOutput: string
  speakSource: Language
  speakTarget: Language
  speakInput: string
  speakOutput: string
  trigger: 'voice' | 'key'
  shortcut: string
}
const storageKey = 'lychee.interpret.v1'
const defaults: Settings = {
  listenSource: 'auto',
  listenTarget: 'zh-CN',
  listenOutput: 'default',
  speakSource: 'zh-CN',
  speakTarget: 'en-US',
  speakInput: 'default',
  speakOutput: 'default',
  trigger: 'voice',
  shortcut: '',
}

function readSettings(): Settings {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey) || 'null')
    if (!saved || typeof saved !== 'object') return defaults
    const language = (value: unknown, fallback: Language): Language =>
      LANGUAGES.find((item) => item.value === value)?.value ?? fallback
    return {
      ...defaults,
      listenSource:
        saved.listenSource === 'auto'
          ? 'auto'
          : language(saved.listenSource, 'zh-CN'),
      listenTarget: language(saved.listenTarget, defaults.listenTarget),
      speakSource: language(saved.speakSource, defaults.speakSource),
      speakTarget: language(saved.speakTarget, defaults.speakTarget),
      trigger: saved.trigger === 'key' ? 'key' : 'voice',
      shortcut: typeof saved.shortcut === 'string' ? saved.shortcut : '',
      ...Object.fromEntries(
        ['listenOutput', 'speakInput', 'speakOutput']
          .filter((key) => typeof saved[key] === 'string' && saved[key])
          .map((key) => [key, saved[key]]),
      ),
    }
  } catch {
    return defaults
  }
}

type Option = { value: string; label: string }

function SettingSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: readonly Option[]
  onChange: (value: string) => void
}) {
  return (
    <Select
      items={options}
      value={value}
      onValueChange={(next) => next !== null && onChange(next)}
    >
      <SelectTrigger
        className={cn(
          "[&[data-slot='select-trigger']]:w-full [&[data-slot='select-trigger']]:h-[30px] [&[data-slot='select-trigger']]:border",
          "[&[data-slot='select-trigger']]:border-[var(--interpret-field-border)] [&[data-slot='select-trigger']]:rounded-[9px]",
          "[&[data-slot='select-trigger']]:bg-[var(--interpret-field)] [&[data-slot='select-trigger']]:text-[color:var(--interpret-ink)]",
          "[&[data-slot='select-trigger']]:text-[13px] [&[data-slot='select-trigger']]:font-normal [&[data-slot='select-trigger']]:py-0",
          "[&[data-slot='select-trigger']]:px-[11px] [&[data-slot='select-trigger']]:shadow-none",
          "[&[data-slot='select-trigger']]:[transition:background_160ms_ease,_border-color_160ms_ease]",
          "[&[data-slot='select-trigger']:hover]:[background:color-mix(_in_srgb,_var(--interpret-field),_var(--interpret-ink)_4%_)]",
          "[&[data-slot='select-trigger']:hover]:border-[color-mix(_in_srgb,_var(--interpret-field-border),_var(--interpret-ink)_12%_)]",
          "[&_[data-slot='select-value']]:block [&_[data-slot='select-value']]:overflow-hidden",
          "[&_[data-slot='select-value']]:text-ellipsis [&_[data-slot='select-value']]:whitespace-nowrap",
          "[&[data-slot='select-trigger']_>_svg]:w-[14px] [&[data-slot='select-trigger']_>_svg]:h-[14px]",
          "[&[data-slot='select-trigger']_>_svg]:text-[color:#728198] [&[data-slot='select-trigger']_>_svg]:[stroke-width:2.5]",
          "[@media(min-height:760px)]:[&[data-slot='select-trigger']]:text-[16px]",
          "[@media(min-height:760px)]:[&[data-slot='select-trigger']]:h-[39px]",
          "[@media(min-height:760px)]:[&[data-slot='select-trigger']]:rounded-[11px]",
        )}
        aria-label={label}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        className="p-1 [&_[data-slot='select-item']]:min-h-[30px] [&_[data-slot='select-item']]:text-[13px] [&_[data-slot='select-item']]:pl-2"
        align="start"
        alignItemWithTrigger={false}
        sideOffset={6}
      >
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function SettingRow({
  icon: Icon,
  label,
  children,
}: {
  icon: LucideIcon
  label: string
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        'grid grid-cols-[58px_minmax(90px,_1fr)_minmax(0,_308px)] items-center gap-x-5 min-h-[var(--interpret-row-height)]',
        '[border-bottom:1px_solid_var(--interpret-line)] [@media(min-height:760px)]:grid-cols-[72px_minmax(110px,_1fr)_minmax(0,_42%)]',
        '[@media(min-height:760px)]:gap-x-6.5 [@media(max-width:700px)]:grid-cols-[38px_75px_minmax(0,_1fr)]!',
        '[@media(max-width:700px)]:gap-x-2.5!',
      )}
    >
      <Icon
        className="justify-self-center w-[23px] h-[23px] text-[color:var(--interpret-ink)]"
        strokeWidth={1.8}
        aria-hidden="true"
      />
      <span className="text-[color:var(--interpret-muted)] text-[13px] font-semibold [@media(min-height:760px)]:text-[16px]">
        {label}
      </span>
      <div className="min-w-0 pt-1 pr-1 pb-1 pl-0">{children}</div>
    </div>
  )
}

const defaultInputs = [{ value: 'default', label: '麦克风' }]
const defaultOutputs = [{ value: 'default', label: '默认扬声器' }]

function useDevices() {
  const [inputs, setInputs] = useState<Option[]>(defaultInputs)
  const [outputs, setOutputs] = useState<Option[]>(defaultOutputs)
  useEffect(() => {
    const media = navigator.mediaDevices
    if (!media?.enumerateDevices) return
    let alive = true
    const refresh = async () => {
      try {
        const devices = await media.enumerateDevices()
        if (!alive) return
        const options = (kind: MediaDeviceKind) =>
          devices
            .filter(
              (device) =>
                device.kind === kind &&
                device.deviceId &&
                device.deviceId !== 'default' &&
                device.label,
            )
            .map((device) => ({ value: device.deviceId, label: device.label }))
        setInputs([...defaultInputs, ...options('audioinput')])
        setOutputs([...defaultOutputs, ...options('audiooutput')])
      } catch {
        // Default devices remain selectable when device enumeration is unavailable.
      }
    }
    void refresh()
    media.addEventListener('devicechange', refresh)
    return () => {
      alive = false
      media.removeEventListener('devicechange', refresh)
    }
  }, [])
  return { inputs, outputs }
}

export function InterpretWorkspace() {
  const [settings, setSettings] = useState(readSettings)
  const [recording, setRecording] = useState(false)
  const shortcutRef = useRef<HTMLButtonElement>(null)
  const shortcutId = useId()
  const reducedMotion = useReducedMotion()
  const devices = useDevices()
  const toast = useToastManager()
  const patch = (changes: Partial<Settings>) =>
    setSettings((previous) => ({ ...previous, ...changes }))
  const withSelected = (options: Option[], selected: string) =>
    options.some((option) => option.value === selected)
      ? options
      : [...options, { value: selected, label: '所选设备当前不可用' }]

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(settings))
    } catch {
      // Settings still work for this session when storage is unavailable.
    }
  }, [settings])

  useEffect(() => {
    if (!recording) return
    function capture(event: KeyboardEvent) {
      if (event.key === 'Tab') {
        setRecording(false)
        return
      }
      event.preventDefault()
      event.stopPropagation()
      if (event.key === 'Escape') {
        setRecording(false)
        return
      }
      if (
        event.repeat ||
        ['Control', 'Alt', 'Shift', 'Meta'].includes(event.key)
      )
        return
      if (event.key === 'Backspace' || event.key === 'Delete') {
        patch({ shortcut: '' })
        setRecording(false)
        return
      }
      if (
        !(event.ctrlKey || event.altKey || event.metaKey) &&
        !/^F([1-9]|1[0-2])$/.test(event.key)
      )
        return
      const key = event.code.startsWith('Key')
        ? event.code.slice(3)
        : event.code.startsWith('Digit')
          ? event.code.slice(5)
          : event.key === ' '
            ? 'Space'
            : event.key
      patch({
        shortcut: [
          event.ctrlKey && 'Ctrl',
          event.altKey && 'Alt',
          event.shiftKey && 'Shift',
          event.metaKey && 'Meta',
          key,
        ]
          .filter(Boolean)
          .join(' + '),
      })
      setRecording(false)
    }
    window.addEventListener('keydown', capture, true)
    return () => window.removeEventListener('keydown', capture, true)
  }, [recording])

  return (
    <section
      className={cn(
        '[--interpret-ink:#102d52] [--interpret-muted:#5e708b] [--interpret-line:#e8edf4] [--interpret-field:#f0f3f7]',
        '[--interpret-field-border:#e2e8f0] [--interpret-row-height:40px] w-[calc(100%_-_92px)] max-w-[1120px] my-0 mx-auto pt-0 px-0',
        'pb-3 text-[color:var(--interpret-ink)] [align-self:start] dark:[--interpret-ink:#d6e5ff] dark:[--interpret-muted:#a5b3cc]',
        'dark:[--interpret-line:#2d3444] dark:[--interpret-field:#242b38] dark:[--interpret-field-border:#343f52]',
        '[@media(min-height:760px)]:[--interpret-row-height:54px] [@media(min-height:760px)]:pt-4',
        '[@media(max-width:700px)]:w-[calc(100%_-_24px)] [@media(prefers-reduced-motion:reduce)]:[&_*]:[transition:none]!',
        '[@media(prefers-reduced-motion:reduce)]:[&_*::before]:[transition:none]!',
      )}
      aria-label="同声传译设置"
    >
      <section
        className="interpret-section"
        aria-labelledby="interpret-listen-title"
      >
        <div
          className={cn(
            'flex items-center gap-5 min-h-[64px] pb-1.5 [border-bottom:1px_solid_var(--interpret-line)] [&_img]:shrink-0 [&_img]:w-[56px]',
            '[&_img]:h-[56px] [&_h1]:mt-0 [&_h1]:mx-0 [&_h1]:mb-0.5 [&_h1]:text-foreground [&_h1]:text-[24px] [&_h1]:font-bold',
            '[&_h1]:leading-[1.25] [&_h1]:tracking-[-0.6px] [&_h2]:mt-0 [&_h2]:mx-0 [&_h2]:mb-0.5 [&_h2]:text-foreground [&_h2]:text-[24px]',
            '[&_h2]:font-bold [&_h2]:leading-[1.25] [&_h2]:tracking-[-0.6px] [&_p]:flex [&_p]:items-center [&_p]:gap-2 [&_p]:m-0',
            '[&_p]:text-[color:var(--interpret-muted)] [&_p]:text-[14px] [&_p]:font-semibold [&_p_svg]:w-[17px] [&_p_svg]:h-[17px]',
            '[@media(min-height:760px)]:min-h-[84px] [@media(min-height:760px)]:gap-6.5 [@media(min-height:760px)]:pb-3',
            '[@media(min-height:760px)]:[&_img]:w-[72px] [@media(min-height:760px)]:[&_img]:h-[72px]',
            '[@media(min-height:760px)]:[&_h1]:text-[30px] [@media(min-height:760px)]:[&_h2]:text-[30px]',
            '[@media(min-height:760px)]:[&_p]:text-[17px]',
          )}
        >
          <img
            src={listenIcon}
            width={58}
            height={58}
            alt=""
            draggable={false}
          />
          <div>
            <h1 id="interpret-listen-title">听别人</h1>
            <p>
              系统声音 <ArrowRight aria-hidden="true" /> 我的语言
            </p>
          </div>
        </div>
        <SettingRow icon={Globe} label="语言">
          <div className="grid grid-cols-[minmax(0,_1fr)_20px_minmax(0,_1.04fr)] items-center gap-[7px]">
            <SettingSelect
              label="听别人 · 原语言"
              value={settings.listenSource}
              options={SOURCE_LANGUAGES}
              onChange={(value) =>
                patch({ listenSource: value as SourceLanguage })
              }
            />
            <ArrowRight
              className="w-[18px] h-[18px] justify-self-center text-[color:var(--interpret-muted)]"
              aria-hidden="true"
            />
            <SettingSelect
              label="听别人 · 我的语言"
              value={settings.listenTarget}
              options={LANGUAGES}
              onChange={(value) => patch({ listenTarget: value as Language })}
            />
          </div>
        </SettingRow>
        <SettingRow icon={Volume2} label="输出设备">
          <SettingSelect
            label="听别人 · 输出设备"
            value={settings.listenOutput}
            options={withSelected(devices.outputs, settings.listenOutput)}
            onChange={(listenOutput) => patch({ listenOutput })}
          />
        </SettingRow>
      </section>

      <section
        className="interpret-section mt-7 [@media(min-height:760px)]:mt-10.5"
        aria-labelledby="interpret-speak-title"
      >
        <div
          className={cn(
            'flex items-center gap-5 min-h-[64px] pb-1.5 [border-bottom:1px_solid_var(--interpret-line)] [&_img]:shrink-0 [&_img]:w-[56px]',
            '[&_img]:h-[56px] [&_h1]:mt-0 [&_h1]:mx-0 [&_h1]:mb-0.5 [&_h1]:text-foreground [&_h1]:text-[24px] [&_h1]:font-bold',
            '[&_h1]:leading-[1.25] [&_h1]:tracking-[-0.6px] [&_h2]:mt-0 [&_h2]:mx-0 [&_h2]:mb-0.5 [&_h2]:text-foreground [&_h2]:text-[24px]',
            '[&_h2]:font-bold [&_h2]:leading-[1.25] [&_h2]:tracking-[-0.6px] [&_p]:flex [&_p]:items-center [&_p]:gap-2 [&_p]:m-0',
            '[&_p]:text-[color:var(--interpret-muted)] [&_p]:text-[14px] [&_p]:font-semibold [&_p_svg]:w-[17px] [&_p_svg]:h-[17px]',
            '[@media(min-height:760px)]:min-h-[84px] [@media(min-height:760px)]:gap-6.5 [@media(min-height:760px)]:pb-3',
            '[@media(min-height:760px)]:[&_img]:w-[72px] [@media(min-height:760px)]:[&_img]:h-[72px]',
            '[@media(min-height:760px)]:[&_h1]:text-[30px] [@media(min-height:760px)]:[&_h2]:text-[30px]',
            '[@media(min-height:760px)]:[&_p]:text-[17px]',
          )}
        >
          <img
            src={speakIcon}
            width={58}
            height={58}
            alt=""
            draggable={false}
          />
          <div>
            <h2 id="interpret-speak-title">帮我说</h2>
            <p>
              我的语言 <ArrowRight aria-hidden="true" /> 目标语言
            </p>
          </div>
        </div>
        <SettingRow icon={Mic} label="输入来源">
          <SettingSelect
            label="帮我说 · 输入来源"
            value={settings.speakInput}
            options={withSelected(devices.inputs, settings.speakInput)}
            onChange={(speakInput) => patch({ speakInput })}
          />
        </SettingRow>
        <SettingRow icon={Globe} label="语言">
          <div className="grid grid-cols-[minmax(0,_1fr)_20px_minmax(0,_1.04fr)] items-center gap-[7px]">
            <SettingSelect
              label="帮我说 · 我的语言"
              value={settings.speakSource}
              options={LANGUAGES}
              onChange={(value) => patch({ speakSource: value as Language })}
            />
            <ArrowRight
              className="w-[18px] h-[18px] justify-self-center text-[color:var(--interpret-muted)]"
              aria-hidden="true"
            />
            <SettingSelect
              label="帮我说 · 目标语言"
              value={settings.speakTarget}
              options={LANGUAGES}
              onChange={(value) => patch({ speakTarget: value as Language })}
            />
          </div>
        </SettingRow>
        <SettingRow icon={Volume2} label="输出设备">
          <SettingSelect
            label="帮我说 · 输出设备"
            value={settings.speakOutput}
            options={withSelected(devices.outputs, settings.speakOutput)}
            onChange={(speakOutput) => patch({ speakOutput })}
          />
        </SettingRow>
        <SettingRow icon={Zap} label="触发方式">
          <div
            className={cn(
              'w-full h-[30px] border border-[var(--interpret-field-border)] rounded-[9px] bg-[var(--interpret-field)]',
              'text-[color:var(--interpret-ink)] text-[13px] font-normal relative flex isolate p-[1px] [&_button]:flex-1 [&_button]:min-w-0',
              '[&_button]:py-0 [&_button]:px-2 [&_button]:border-0 [&_button]:rounded-[8px] [&_button]:bg-transparent',
              '[&_button]:text-[color:var(--interpret-ink)] [&_button]:[font:inherit] [&_button]:[transition:color_160ms_ease]',
              "[&_button[aria-checked='true']]:text-white dark:[&_button[aria-checked='true']]:text-[color:#10233e]",
              '[@media(min-height:760px)]:text-[16px] [@media(min-height:760px)]:h-[39px] [@media(min-height:760px)]:rounded-[11px]',
            )}
            role="radiogroup"
            aria-label="触发方式"
          >
            <motion.span
              className="absolute z-[-1] [top:0] [bottom:0] [left:0] w-[50%] rounded-[9px] [background:linear-gradient(125deg,_#252525,_#050505_80%)] shadow-[0_1px_3px_#0000001a] dark:[background:linear-gradient(125deg,_#e1e8f3,_#fff)]"
              initial={false}
              animate={{ x: settings.trigger === 'voice' ? '0%' : '100%' }}
              transition={
                reducedMotion
                  ? { duration: 0 }
                  : { type: 'spring', stiffness: 460, damping: 38 }
              }
            />
            {(['voice', 'key'] as const).map((trigger) => (
              <button
                key={trigger}
                className="cursor-pointer focus-visible:[outline:2px_solid_var(--color-ring)] focus-visible:outline-offset-2"
                type="button"
                role="radio"
                aria-checked={settings.trigger === trigger}
                aria-controls={shortcutId}
                tabIndex={settings.trigger === trigger ? 0 : -1}
                onClick={() => {
                  patch({ trigger })
                  setRecording(false)
                }}
                onKeyDown={(event) => {
                  if (
                    ![
                      'ArrowLeft',
                      'ArrowRight',
                      'ArrowUp',
                      'ArrowDown',
                      'Home',
                      'End',
                    ].includes(event.key)
                  )
                    return
                  event.preventDefault()
                  const next =
                    event.key === 'Home'
                      ? 'voice'
                      : event.key === 'End'
                        ? 'key'
                        : trigger === 'voice'
                          ? 'key'
                          : 'voice'
                  patch({ trigger: next })
                  setRecording(false)
                  const buttons =
                    event.currentTarget.parentElement?.querySelectorAll(
                      'button',
                    )
                  buttons?.[next === 'voice' ? 0 : 1]?.focus()
                }}
              >
                {trigger === 'voice' ? '语音活动' : '按键触发'}
              </button>
            ))}
          </div>
        </SettingRow>
        <AnimatePresence initial={false}>
          {settings.trigger === 'key' && (
            <motion.div
              id={shortcutId}
              key="shortcut"
              className="overflow-hidden"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={
                reducedMotion
                  ? { duration: 0 }
                  : {
                      height: { duration: 0.32, ease: [0.22, 1, 0.36, 1] },
                      opacity: { duration: 0.18 },
                    }
              }
            >
              <SettingRow icon={Keyboard} label="快捷键">
                <button
                  ref={shortcutRef}
                  type="button"
                  className={cn(
                    'w-full h-[30px] border border-[var(--interpret-field-border)] rounded-[9px] bg-[var(--interpret-field)]',
                    'text-[color:var(--interpret-ink)] text-[13px] font-normal',
                    'hover:[background:color-mix(_in_srgb,_var(--interpret-field),_var(--interpret-ink)_4%_)]',
                    'hover:border-[color-mix(_in_srgb,_var(--interpret-field-border),_var(--interpret-ink)_12%_)] py-0 px-3',
                    "[transition:background_160ms_ease,_border-color_160ms_ease] [&[data-recording='true']]:border-[#168bf2]",
                    "[&[data-recording='true']]:bg-[#eaf5ff] [&[data-recording='true']]:text-[color:#1374c5] [@media(min-height:760px)]:text-[16px]",
                    '[@media(min-height:760px)]:h-[39px] [@media(min-height:760px)]:rounded-[11px] cursor-pointer',
                    'focus-visible:[outline:2px_solid_var(--color-ring)] focus-visible:outline-offset-2',
                  )}
                  data-recording={recording}
                  aria-label={
                    recording
                      ? '请按组合键，Esc 取消'
                      : `自定义快捷键${settings.shortcut ? `：${settings.shortcut}` : ''}`
                  }
                  aria-describedby={
                    recording ? `${shortcutId}-hint` : undefined
                  }
                  onClick={() => setRecording((previous) => !previous)}
                  onBlur={() => setRecording(false)}
                  title="点击录入 Ctrl / Alt + 按键，或 F1–F12；Esc 取消，Delete 清除"
                >
                  {recording ? '请按组合键…' : settings.shortcut || '自定义'}
                </button>
              </SettingRow>
              {recording && (
                <span
                  className="sr-only"
                  id={`${shortcutId}-hint`}
                  role="status"
                >
                  请按 Ctrl 或 Alt 加一个按键，也可使用 F1 到 F12。Esc
                  取消，Delete 清除。
                </span>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </section>

      <div className="flex justify-center pt-3.5 [@media(min-height:760px)]:pt-5">
        <button
          type="button"
          className={cn(
            'inline-flex justify-center items-center gap-3.5 w-[210px] h-[40px] py-0 px-6 border border-[#151515] rounded-[99px]',
            '[background:linear-gradient(125deg,_#282828,_#030303_80%)] shadow-[0_8px_22px_#18263e0a] text-white text-[15px] font-normal',
            '[transition:transform_160ms_ease,_box-shadow_160ms_ease] [&_svg]:w-[24px] [&_svg]:h-[24px] hover:shadow-[0_6px_18px_#18263e24]',
            'hover:[transform:translateY(-1px)] active:[transform:translateY(0)_scale(0.98)]',
            'dark:[background:linear-gradient(125deg,_#e1e8f3,_#fff)] dark:text-[color:#10233e] [@media(min-height:760px)]:w-[264px]',
            '[@media(min-height:760px)]:h-[51px] [@media(min-height:760px)]:text-[18px] cursor-pointer',
            'focus-visible:[outline:2px_solid_var(--color-ring)] focus-visible:outline-offset-2',
          )}
          onClick={() => {
            if (settings.trigger === 'key' && !settings.shortcut) {
              shortcutRef.current?.focus()
              setRecording(true)
              return
            }
            toast.add({
              id: 'interpret-notice',
              title: '同传配置已保存，双向语音同传服务尚未接入',
            })
          }}
        >
          <AudioLines aria-hidden="true" strokeWidth={1.8} />
          开始同传
        </button>
      </div>
    </section>
  )
}
