import { cn } from '@/lib/utils'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import {
  ArrowRight,
  AudioLines,
  Ear,
  Mic,
  X,
  type LucideIcon,
} from 'lucide-react'
import {
  PopupSelect,
  SegmentedControl,
  SettingsGroup,
  SettingsRow,
} from '@/components/ui/settings'
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
const TRIGGERS = [
  { value: 'voice', label: '语音活动' },
  { value: 'key', label: '按键触发' },
] as const

function readSettings(): Settings {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey) || 'null')
    if (!saved || typeof saved !== 'object') return defaults
    const language = (value: unknown, fallback: Language): Language =>
      LANGUAGES.find((item) => item.value === value)?.value ?? fallback
    return {
      ...defaults,
      listenSource:
        SOURCE_LANGUAGES.find((item) => item.value === saved.listenSource)
          ?.value ?? defaults.listenSource,
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

/** Card header: an app-style glyph tile, a title and a one-line purpose. */
function GroupHeader({
  id,
  icon: Icon,
  tint,
  title,
  description,
}: {
  id: string
  icon: LucideIcon
  tint: string
  title: string
  description: string
}) {
  return (
    <div className="flex h-[50px] items-center gap-3 px-3.5">
      <span
        className={cn(
          'grid size-[30px] shrink-0 place-items-center rounded-[8px] text-white',
          'shadow-[inset_0_0.5px_0_#ffffff59,_0_1px_2px_#00000026,_0_0_0_0.5px_#0000001a]',
          tint,
        )}
        aria-hidden="true"
      >
        <Icon className="size-[17px]" strokeWidth={2} />
      </span>
      <div className="min-w-0">
        <h2
          id={id}
          className="m-0 text-[14px] leading-5 font-semibold text-[color:var(--ui-label)]"
        >
          {title}
        </h2>
        <p className="m-0 truncate text-[12px] leading-4 text-[color:var(--ui-secondary)]">
          {description}
        </p>
      </div>
    </div>
  )
}

function LanguagePair({ from, to }: { from: ReactNode; to: ReactNode }) {
  return (
    <>
      {from}
      <ArrowRight
        className="size-3.5 shrink-0 text-[color:var(--ui-secondary)]"
        strokeWidth={1.75}
        aria-hidden="true"
      />
      {to}
    </>
  )
}

export function InterpretWorkspace() {
  const [settings, setSettings] = useState(readSettings)
  const [recording, setRecording] = useState(false)
  const shortcutRef = useRef<HTMLButtonElement>(null)
  const shortcutId = useId()
  const devices = useDevices()
  const toast = useToastManager()
  const patch = (changes: Partial<Settings>) =>
    setSettings((previous) => ({ ...previous, ...changes }))
  const withSelected = (options: Option[], selected: string) =>
    options.some((option) => option.value === selected)
      ? options
      : [...options, { value: selected, label: '所选设备当前不可用' }]
  const keys = settings.shortcut ? settings.shortcut.split(' + ') : []

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

  const hint =
    settings.trigger === 'voice'
      ? '说话时自动识别并翻译'
      : settings.shortcut
        ? `按 ${settings.shortcut} 开始说话`
        : '请先设置说话快捷键'

  return (
    <section
      className="grid w-full grid-cols-[296px_minmax(0,_1fr)] gap-4 self-start pt-1 pb-3 [--ui-label-width:3rem] [@media(max-width:760px)]:grid-cols-1"
      aria-label="同声传译设置"
    >
      <div
        className={cn(
          'relative isolate flex min-h-[300px] flex-col items-center justify-center overflow-hidden rounded-[14px] px-6',
          'border border-[var(--ui-group-border)] bg-[var(--ui-group)] shadow-[var(--ui-group-shadow)]',
        )}
      >
        <div
          aria-hidden="true"
          className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_22%_12%,#3b9dff24,transparent_52%),radial-gradient(circle_at_82%_92%,#ff4d5a1c,transparent_52%)]"
        />
        <div className="group relative grid place-items-center">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute -inset-[34px] rounded-full border border-[var(--ui-separator)] bg-[var(--ui-control)]/25 shadow-[inset_0_1px_2px_#0000000a] transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-[1.04]"
          />
          <span
            aria-hidden="true"
            className="pointer-events-none absolute -inset-[16px] rounded-full border border-[var(--ui-group-border)] bg-[var(--ui-control)]/60 shadow-[inset_0_1px_2px_#0000000f,_0_1px_1px_#ffffff80] transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-[1.03] dark:shadow-[inset_0_1px_2px_#00000040]"
          />
          <button
            type="button"
            aria-label="开始同传"
            aria-describedby="interpret-start-hint"
            className={cn(
              'relative grid size-[104px] cursor-pointer place-items-center rounded-full text-white',
              'bg-[linear-gradient(180deg,#3d3d42,#1a1a1d)]',
              'shadow-[inset_0_1px_0_#ffffff38,_0_0_0_0.5px_#00000066,_0_2px_4px_#00000033,_0_12px_24px_-8px_#00000073]',
              'transition-[filter,transform] duration-150 hover:brightness-[1.15] active:scale-[0.97]',
              'dark:bg-[linear-gradient(180deg,#ffffff,#dfe0e7)] dark:text-[#1c1c1f]',
              'dark:shadow-[inset_0_1px_0_#ffffff,_0_0_0_0.5px_#000000a0,_0_2px_4px_#00000066,_0_12px_24px_-8px_#000000cc]',
              'dark:hover:brightness-[1.04] focus-visible:outline-2 focus-visible:outline-offset-[20px] focus-visible:outline-[var(--color-ring)]',
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
            <AudioLines
              className="size-9"
              strokeWidth={1.6}
              aria-hidden="true"
            />
          </button>
        </div>
        <p className="mt-12 mb-0 text-[15px] font-semibold text-[color:var(--ui-label)]">
          开始同传
        </p>
        <p
          id="interpret-start-hint"
          className="mt-1 mb-0 max-w-full truncate text-center text-[12px] text-[color:var(--ui-secondary)]"
        >
          {hint}
        </p>
      </div>

      <div className="flex min-w-0 flex-col gap-4">
        <SettingsGroup labelledBy="interpret-listen-title">
          <GroupHeader
            id="interpret-listen-title"
            icon={Ear}
            tint="bg-[linear-gradient(180deg,#5cb2ff,#1a7cf2)]"
            title="听别人"
            description="把电脑播放的声音译成我的语言"
          />
          <SettingsRow label="语言">
            <LanguagePair
              from={
                <PopupSelect
                  label="听别人 · 原语言"
                  value={settings.listenSource}
                  items={SOURCE_LANGUAGES}
                  onChange={(value) =>
                    patch({ listenSource: value as SourceLanguage })
                  }
                />
              }
              to={
                <PopupSelect
                  label="听别人 · 我的语言"
                  value={settings.listenTarget}
                  items={LANGUAGES}
                  onChange={(value) =>
                    patch({ listenTarget: value as Language })
                  }
                />
              }
            />
          </SettingsRow>
          <SettingsRow label="输出">
            <PopupSelect
              label="听别人 · 输出设备"
              value={settings.listenOutput}
              items={withSelected(devices.outputs, settings.listenOutput)}
              onChange={(listenOutput) => patch({ listenOutput })}
            />
          </SettingsRow>
        </SettingsGroup>

        <SettingsGroup labelledBy="interpret-speak-title">
          <GroupHeader
            id="interpret-speak-title"
            icon={Mic}
            tint="bg-[linear-gradient(180deg,#ff7a7f,#ee3640)]"
            title="帮我说"
            description="把我说的话译成对方的语言"
          />
          <SettingsRow label="输入">
            <PopupSelect
              label="帮我说 · 输入来源"
              value={settings.speakInput}
              items={withSelected(devices.inputs, settings.speakInput)}
              onChange={(speakInput) => patch({ speakInput })}
            />
          </SettingsRow>
          <SettingsRow label="语言">
            <LanguagePair
              from={
                <PopupSelect
                  label="帮我说 · 我的语言"
                  value={settings.speakSource}
                  items={LANGUAGES}
                  onChange={(value) =>
                    patch({ speakSource: value as Language })
                  }
                />
              }
              to={
                <PopupSelect
                  label="帮我说 · 目标语言"
                  value={settings.speakTarget}
                  items={LANGUAGES}
                  onChange={(value) =>
                    patch({ speakTarget: value as Language })
                  }
                />
              }
            />
          </SettingsRow>
          <SettingsRow label="输出">
            <PopupSelect
              label="帮我说 · 输出设备"
              value={settings.speakOutput}
              items={withSelected(devices.outputs, settings.speakOutput)}
              onChange={(speakOutput) => patch({ speakOutput })}
            />
          </SettingsRow>
          <SettingsRow label="触发">
            <SegmentedControl
              name="interpret-trigger"
              label="触发方式"
              value={settings.trigger}
              options={TRIGGERS}
              onChange={(trigger) => {
                patch({ trigger })
                setRecording(false)
              }}
            />
          </SettingsRow>
          <SettingsRow label="快捷键" htmlFor={shortcutId}>
            <button
              id={shortcutId}
              ref={shortcutRef}
              type="button"
              className={cn(
                'flex h-[28px] min-w-0 flex-1 cursor-pointer items-center gap-1 rounded-[7px] border border-[var(--ui-control-border)] px-2',
                'bg-[var(--ui-control)] text-[13px] text-[color:var(--ui-label)] shadow-[var(--ui-control-shadow)]',
                'transition-[background-color,box-shadow] duration-150 hover:bg-[var(--ui-control-hover)]',
                'focus-visible:outline-2 focus-visible:outline-[var(--color-ring)] disabled:cursor-default disabled:opacity-45 disabled:hover:bg-[var(--ui-control)]',
                recording &&
                  'border-[#0a84ff] shadow-[0_0_0_3px_color-mix(in_srgb,#0a84ff_22%,transparent)]',
              )}
              aria-label={
                recording
                  ? '请按组合键，Esc 取消'
                  : `自定义快捷键${settings.shortcut ? `：${settings.shortcut}` : ''}`
              }
              aria-describedby={recording ? `${shortcutId}-hint` : undefined}
              disabled={settings.trigger !== 'key'}
              onClick={() => setRecording((previous) => !previous)}
              onBlur={() => setRecording(false)}
              title="点击录入 Ctrl / Alt + 按键，或 F1–F12；Esc 取消，Delete 清除"
            >
              {recording || !keys.length ? (
                <span className="px-0.5 text-[color:var(--ui-secondary)]">
                  {recording ? '请按组合键…' : '点击录入'}
                </span>
              ) : (
                keys.map((key) => (
                  <kbd
                    key={key}
                    className="inline-grid h-[19px] min-w-[19px] place-items-center rounded-[4px] border border-[var(--ui-control-border)] bg-[var(--ui-group)] px-1.5 font-[inherit] text-[11px] font-medium shadow-[0_1px_0_var(--ui-control-border)]"
                  >
                    {key}
                  </kbd>
                ))
              )}
            </button>
            {settings.shortcut && !recording && settings.trigger === 'key' && (
              <button
                type="button"
                className="grid size-6 shrink-0 cursor-pointer place-items-center rounded-[6px] text-[color:var(--ui-secondary)] transition-colors hover:bg-[var(--ui-track)] hover:text-[color:var(--ui-label)] focus-visible:outline-2 focus-visible:outline-[var(--color-ring)]"
                aria-label="清除快捷键"
                title="清除快捷键"
                onClick={() => patch({ shortcut: '' })}
              >
                <X className="size-3.5" strokeWidth={1.75} />
              </button>
            )}
          </SettingsRow>
          {recording && (
            <span className="sr-only" id={`${shortcutId}-hint`} role="status">
              请按 Ctrl 或 Alt 加一个按键，也可使用 F1 到 F12。Esc 取消，Delete
              清除。
            </span>
          )}
        </SettingsGroup>
      </div>
    </section>
  )
}
