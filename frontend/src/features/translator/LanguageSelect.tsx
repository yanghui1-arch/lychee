import { useId } from 'react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LANGUAGES, type Language } from './model'

export function LanguageSelect({
  label,
  value,
  onChange,
}: {
  label: string
  value: Language
  onChange: (value: Language) => void
}) {
  const id = useId()
  return (
    <Select
      items={LANGUAGES}
      value={value}
      onValueChange={(next) => {
        if (next) onChange(next)
      }}
    >
      <label className="language-label" htmlFor={id}>
        {label}
      </label>
      <SelectTrigger id={id} className="language-trigger" aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        className="select-popup"
        align="start"
        sideOffset={8}
        alignItemWithTrigger={false}
      >
        {LANGUAGES.map((language) => (
          <SelectItem
            className="select-item"
            key={language.value}
            value={language.value}
          >
            {language.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
