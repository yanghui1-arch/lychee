import { describe, expect, test } from 'bun:test'
import {
  defaultSubtitleSettings,
  readSubtitleSettings,
  subtitleStorageKey,
} from '../src/features/subtitle/model'

function storage(current: string | null, legacy: string | null = null) {
  return {
    getItem: (key: string) => (key === subtitleStorageKey ? current : legacy),
  }
}

describe('subtitle settings compatibility', () => {
  const preferences = {
    source: 'fr-FR',
    target: 'ja-JP',
    audio: 'app:123:456',
    fontSize: 40,
    position: 'custom',
    customX: 0,
    customY: 100,
    style: 'paper',
  }

  test('preserves a complete current configuration', () => {
    expect(readSubtitleSettings(storage(JSON.stringify(preferences)))).toEqual(
      preferences,
    )
  })

  test('v1 migration resets only the source to global listening', () => {
    expect(
      readSubtitleSettings(storage(null, JSON.stringify(preferences))),
    ).toEqual({ ...preferences, audio: 'system' })
  })

  test('current settings take precedence over legacy settings', () => {
    const current = { ...preferences, style: 'slate', fontSize: 18 }
    expect(
      readSubtitleSettings(
        storage(JSON.stringify(current), JSON.stringify(preferences)),
      ),
    ).toEqual(current)
  })

  test('invalid fields fall back without discarding valid fields', () => {
    const invalid = {
      source: 'invalid',
      target: 'ko-KR',
      audio: 'microphone',
      fontSize: '40',
      position: 'custom',
      customX: null,
      customY: 105,
      style: 'invalid',
    }
    expect(readSubtitleSettings(storage(JSON.stringify(invalid)))).toEqual({
      ...defaultSubtitleSettings,
      target: 'ko-KR',
      position: 'custom',
    })
  })

  test('corrupt or unavailable storage keeps the workspace usable', () => {
    for (const raw of [null, '{', 'null', 'false', '3', '[]']) {
      expect(readSubtitleSettings(storage(raw))).toEqual(
        defaultSubtitleSettings,
      )
    }
    expect(
      readSubtitleSettings({
        getItem: () => {
          throw new Error('storage unavailable')
        },
      }),
    ).toEqual(defaultSubtitleSettings)
  })

  test('rejects obsolete sources and malformed application identities', () => {
    for (const audio of ['microphone', 'app:1:', 'app:abc:2', 'app:1:2:3']) {
      expect(
        readSubtitleSettings(storage(JSON.stringify({ audio }))).audio,
      ).toBe('system')
    }
  })

  test('rejects fractional font sizes and overflowing positions', () => {
    for (const fontSize of [17, 41, 30.5, null]) {
      expect(
        readSubtitleSettings(storage(JSON.stringify({ fontSize }))).fontSize,
      ).toBe(30)
    }
    const result = readSubtitleSettings(
      storage('{"customX":1e309,"customY":-1e309}'),
    )
    expect([result.customX, result.customY]).toEqual([50, 82])
  })
})
