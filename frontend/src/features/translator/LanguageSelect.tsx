import { useId } from 'react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  LANGUAGES,
  SOURCE_LANGUAGES,
  type Language,
  type SourceLanguage,
} from './model'

type LanguageSelectProps = {
  label: string
  disabled?: boolean
} & (
  | { value: Language; onChange: (value: Language) => void; autoDetect?: false }
  | {
      value: SourceLanguage
      onChange: (value: SourceLanguage) => void
      autoDetect: true
    }
)

export function LanguageSelect(props: LanguageSelectProps) {
  const id = useId()
  const languages = props.autoDetect ? SOURCE_LANGUAGES : LANGUAGES
  return (
    <Select
      disabled={props.disabled}
      items={languages}
      value={props.value}
      onValueChange={(next) => {
        if (next === null) return
        if (props.autoDetect) props.onChange(next)
        else if (next !== 'auto') props.onChange(next)
      }}
    >
      <label className="language-label" htmlFor={id}>
        {props.label}
      </label>
      <SelectTrigger
        id={id}
        className="language-trigger"
        aria-label={props.label}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        className="select-popup"
        align="start"
        sideOffset={8}
        alignItemWithTrigger={false}
      >
        {languages.map((language) => (
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
