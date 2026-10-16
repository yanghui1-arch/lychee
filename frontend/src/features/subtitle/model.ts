import { LANGUAGES, type Language } from '../translator/model'

export const subtitleSamples: Record<Language, string> = {
  'zh-CN': '未来属于那些不断探索的人。',
  'zh-TW': '未來屬於那些不斷探索的人。',
  'en-US': 'The future belongs to those who keep exploring.',
  'en-GB': 'The future belongs to those who keep exploring.',
  'ja-JP': '未来は、探求し続ける人のものです。',
  'ko-KR': '미래는 끊임없이 탐구하는 사람들의 것입니다.',
  'fr-FR': 'L’avenir appartient à ceux qui continuent d’explorer.',
  'de-DE': 'Die Zukunft gehört denen, die immer weiter entdecken.',
}

export const subtitleStyles = [
  { value: 'charcoal', label: '深底白字 · 70% 底色' },
  { value: 'mist', label: '轻雾白字 · 10% 轻底' },
  { value: 'paper', label: '浅底深字 · 15% 轻底' },
  { value: 'slate', label: '暖黄描边 · 无背景' },
  { value: 'graphite', label: '暗夜轻底 · 20% 轻底' },
] as const

export type SubtitlePreferences = {
  source: Language
  target: Language
  audio: string
  fontSize: number
  position: 'top' | 'bottom' | 'custom'
  customX: number
  customY: number
  style: (typeof subtitleStyles)[number]['value']
}

export const defaultSubtitleSettings: SubtitlePreferences = {
  source: 'en-US',
  target: 'zh-CN',
  audio: 'system',
  fontSize: 30,
  position: 'bottom',
  customX: 50,
  customY: 82,
  style: 'charcoal',
}

export const subtitleStorageKey = 'lychee.subtitle.v2'
const legacyStorageKey = 'lychee.subtitle.v1'

function isLanguage(value: unknown): value is Language {
  return LANGUAGES.some((language) => language.value === value)
}

function isAudioSource(value: unknown): value is string {
  return (
    value === 'system' ||
    (typeof value === 'string' && /^app:\d+:\d+$/.test(value))
  )
}

function isPercentage(value: unknown): value is number {
  return typeof value === 'number' && value >= 0 && value <= 100
}

function isStyle(value: unknown): value is SubtitlePreferences['style'] {
  return subtitleStyles.some((style) => style.value === value)
}

function isPosition(value: unknown): value is SubtitlePreferences['position'] {
  return value === 'top' || value === 'bottom' || value === 'custom'
}

export function readSubtitleSettings(
  storage?: Pick<Storage, 'getItem'>,
): SubtitlePreferences {
  const defaults = defaultSubtitleSettings
  try {
    const savedStorage = storage ?? localStorage
    const current = savedStorage.getItem(subtitleStorageKey)
    const saved = JSON.parse(
      current || savedStorage.getItem(legacyStorageKey) || 'null',
    )
    if (!saved || typeof saved !== 'object') return defaults

    // The v1 migration resets only the audio source to global listening.
    return {
      source: isLanguage(saved.source) ? saved.source : defaults.source,
      target: isLanguage(saved.target) ? saved.target : defaults.target,
      audio:
        current && isAudioSource(saved.audio) ? saved.audio : defaults.audio,
      fontSize:
        Number.isInteger(saved.fontSize) &&
        saved.fontSize >= 18 &&
        saved.fontSize <= 40
          ? saved.fontSize
          : defaults.fontSize,
      position: isPosition(saved.position) ? saved.position : defaults.position,
      customX: isPercentage(saved.customX) ? saved.customX : defaults.customX,
      customY: isPercentage(saved.customY) ? saved.customY : defaults.customY,
      style: isStyle(saved.style) ? saved.style : defaults.style,
    }
  } catch {
    return defaults
  }
}
