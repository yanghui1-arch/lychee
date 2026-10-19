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
      <label
        className="language-label shrink-0 text-[13px] text-[color:var(--subtle-foreground)]"
        htmlFor={id}
      >
        {props.label}
      </label>
      <SelectTrigger
        id={id}
        className="language-trigger data-[size=default]:h-[30px] dark:bg-background dark:hover:bg-background w-[144px] h-[30px] py-0 px-[13px] border border-border rounded-[12px] bg-background shadow-none text-[13px] gap-3.5 font-normal [&_svg]:w-[12px] [&_svg]:h-[12px] [&_svg]:text-[color:var(--subtle-foreground)]"
        aria-label={props.label}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        className="text-[12px] border-border rounded-[12px] [animation-duration:var(--motion-duration)] [animation-timing-function:var(--motion-ease)] data-ending-style:[animation-duration:var(--motion-close-duration)]"
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
