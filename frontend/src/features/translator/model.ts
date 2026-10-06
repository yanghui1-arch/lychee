export const SAMPLE_TEXT =
  '荔枝翻译是一款专注于高质量翻译的桌面应用，\n支持文本、语音、文档等多种翻译方式。\n它设计简洁、体验流畅，帮助你轻松跨越语言的边界。'
export const RESULT_TEXT = '翻译结果在这里'
export const MAX_LENGTH = 5000
export const LANGUAGES = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'en-US', label: 'English' },
  { value: 'en-GB', label: 'English (UK)' },
  { value: 'ja-JP', label: '日本語' },
  { value: 'ko-KR', label: '한국어' },
  { value: 'fr-FR', label: 'Français' },
  { value: 'de-DE', label: 'Deutsch' },
] as const

export type Language = (typeof LANGUAGES)[number]['value']
export type Page = 'translate' | 'interpret' | 'settings' | 'manual'
export type Collection = 'history' | 'favorites'
export type Entry = {
  id: string
  text: string
  result?: string
  source: Language
  target: Language
  createdAt: string
}
export type Preferences = { textSize: number; saveHistory: boolean }
export function matchesEntry(
  a: Pick<Entry, 'text' | 'source' | 'target'>,
  b: Pick<Entry, 'text' | 'source' | 'target'>,
) {
  return a.text === b.text && a.source === b.source && a.target === b.target
}

// Static pairs demonstrate the record list; no translation service is called.
export const DEMO_ENTRIES: Entry[] = (
  [
    {
      text: SAMPLE_TEXT,
      result:
        'Litchi Translate is a desktop app focused on high-quality translation.',
      source: 'zh-CN',
      target: 'en-US',
    },
    {
      text: '人工智能正在深刻改变我们的工作和生活方式。',
      result:
        'Artificial intelligence is profoundly changing the way we work and live.',
      source: 'zh-CN',
      target: 'en-US',
    },
    {
      text: 'The future belongs to those who keep learning.',
      result: '未来属于那些不断学习的人。',
      source: 'en-US',
      target: 'zh-CN',
    },
    {
      text: 'ありがとうございます。',
      result: 'Thank you very much.',
      source: 'ja-JP',
      target: 'en-US',
    },
    {
      text: 'Ce monde est plus beau parce que tu es là.',
      result: 'This world is more beautiful because you are here.',
      source: 'fr-FR',
      target: 'en-US',
    },
  ] satisfies Omit<Entry, 'id' | 'createdAt'>[]
).map((entry, index) => ({
  ...entry,
  id: `demo-${index}`,
  createdAt: new Date(Date.now() - index * 36_000_000).toISOString(),
}))
