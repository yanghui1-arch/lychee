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
import './interpret.css'

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
      <SelectTrigger className="interpret-select" aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        className="interpret-select-popup"
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
    <div className="interpret-row">
      <Icon
        className="interpret-row-icon"
        strokeWidth={1.8}
        aria-hidden="true"
      />
      <span className="interpret-row-label">{label}</span>
      <div className="interpret-row-control">{children}</div>
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
    <section className="interpret-workspace" aria-label="同声传译设置">
      <section
        className="interpret-section"
        aria-labelledby="interpret-listen-title"
      >
        <div className="interpret-section-header">
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
          <div className="interpret-language-pair">
            <SettingSelect
              label="听别人 · 原语言"
              value={settings.listenSource}
              options={SOURCE_LANGUAGES}
              onChange={(value) =>
                patch({ listenSource: value as SourceLanguage })
              }
            />
            <ArrowRight
              className="interpret-language-arrow"
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
        className="interpret-section interpret-speak"
        aria-labelledby="interpret-speak-title"
      >
        <div className="interpret-section-header">
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
          <div className="interpret-language-pair">
            <SettingSelect
              label="帮我说 · 我的语言"
              value={settings.speakSource}
              options={LANGUAGES}
              onChange={(value) => patch({ speakSource: value as Language })}
            />
            <ArrowRight
              className="interpret-language-arrow"
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
            className="interpret-trigger"
            role="radiogroup"
            aria-label="触发方式"
          >
            <motion.span
              className="interpret-trigger-pill"
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
              className="interpret-shortcut-reveal"
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
                  className="interpret-shortcut"
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

      <div className="interpret-footer">
        <button
          type="button"
          className="interpret-start"
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
