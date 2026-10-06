import { useEffect, useState } from 'react'
import {
  DEMO_ENTRIES,
  LANGUAGES,
  MAX_LENGTH,
  RESULT_TEXT,
  matchesEntry,
  type Collection,
  type Entry,
  type Language,
  type Preferences,
  type SourceLanguage,
} from './model'

type SavedState = {
  text: string
  source: SourceLanguage
  target: Language
  favorites: Entry[]
  history: Entry[]
  preferences: Preferences
}
const STORAGE_KEY = 'lychee.workspace.v1'
const defaults: SavedState = {
  text: '',
  source: 'auto',
  target: 'en-US',
  favorites: [],
  history: DEMO_ENTRIES,
  preferences: { textSize: 16, saveHistory: true },
}

function readState(): SavedState {
  try {
    const saved: unknown = JSON.parse(
      localStorage.getItem(STORAGE_KEY) || 'null',
    )
    if (!saved || typeof saved !== 'object') return defaults
    const value = saved as Partial<SavedState>
    const validLanguage = (language: unknown): language is Language =>
      LANGUAGES.some((item) => item.value === language)
    const validSource = (language: unknown): language is SourceLanguage =>
      language === 'auto' || validLanguage(language)
    const validEntries = (entries: unknown): entries is Entry[] =>
      Array.isArray(entries) &&
      entries.every(
        (entry) =>
          entry &&
          typeof entry.id === 'string' &&
          typeof entry.text === 'string' &&
          Number.isFinite(Date.parse(entry.createdAt)) &&
          (entry.result === undefined || typeof entry.result === 'string') &&
          validSource(entry.source) &&
          validLanguage(entry.target),
      )
    return {
      text:
        typeof value.text === 'string'
          ? value.text.slice(0, MAX_LENGTH)
          : defaults.text,
      source: validSource(value.source) ? value.source : defaults.source,
      target: validLanguage(value.target) ? value.target : defaults.target,
      favorites: validEntries(value.favorites)
        ? value.favorites.slice(0, 50)
        : [],
      history: validEntries(value.history) ? value.history.slice(0, 20) : [],
      preferences: {
        textSize: [14, 16, 18].includes(value.preferences?.textSize ?? 0)
          ? value.preferences!.textSize
          : 16,
        saveHistory: value.preferences?.saveHistory !== false,
      },
    }
  } catch {
    return defaults
  }
}

export function useTranslator() {
  const [state, setState] = useState(readState)
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    } catch {
      /* Keep the session usable when local storage is unavailable. */
    }
  }, [state])
  const patch = (changes: Partial<SavedState>) =>
    setState((previous) => ({ ...previous, ...changes }))
  const isFavorite = (entry: Entry) =>
    state.favorites.some((item) => matchesEntry(item, entry))

  function saveHistory() {
    if (!state.text.trim() || !state.preferences.saveHistory) return
    const entry: Entry = {
      id: crypto.randomUUID(),
      text: state.text,
      result: RESULT_TEXT,
      source: state.source,
      target: state.target,
      createdAt: new Date().toISOString(),
    }
    setState((previous) => ({
      ...previous,
      history: [
        entry,
        ...previous.history.filter((item) => !matchesEntry(item, entry)),
      ].slice(0, 20),
    }))
  }
  function toggleFavorite(entry: Entry) {
    setState((previous) => ({
      ...previous,
      favorites: previous.favorites.some((item) => matchesEntry(item, entry))
        ? previous.favorites.filter((item) => !matchesEntry(item, entry))
        : [entry, ...previous.favorites].slice(0, 50),
    }))
  }
  return {
    ...state,
    patch,
    saveHistory,
    isFavorite,
    toggleFavorite,
    restore: (entry: Entry) =>
      patch({ text: entry.text, source: entry.source, target: entry.target }),
    removeEntry: (id: string, collection: Collection) =>
      setState((previous) => ({
        ...previous,
        [collection]: previous[collection].filter((entry) => entry.id !== id),
      })),
  }
}
