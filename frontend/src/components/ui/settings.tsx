import { cn } from '@/lib/utils'
import type { ReactNode } from 'react'
import { ChevronsUpDown } from 'lucide-react'
import { motion } from 'motion/react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// Grouped settings controls in the style of native desktop preference panes.
// Colors come from the --ui-* tokens in styles/globals.css.

type Option = { value: string; label: string }

/** A rounded card whose direct children are separated by inset hairlines. */
export function SettingsGroup({
  label,
  labelledBy,
  className,
  children,
}: {
  label?: string
  labelledBy?: string
  className?: string
  children: ReactNode
}) {
  return (
    <section
      aria-label={label}
      aria-labelledby={labelledBy}
      className={cn(
        'min-w-0 rounded-[12px] border border-[var(--ui-group-border)] bg-[var(--ui-group)] py-0.5 shadow-[var(--ui-group-shadow)]',
        '[&>*]:relative [&>*+*]:before:absolute [&>*+*]:before:inset-x-3.5 [&>*+*]:before:top-0 [&>*+*]:before:h-px',
        "[&>*+*]:before:bg-[var(--ui-separator)] [&>*+*]:before:content-['']",
        className,
      )}
    >
      {children}
    </section>
  )
}

/** A labelled row; --ui-label-width and --ui-row-height on an ancestor resize it. */
export function SettingsRow({
  label,
  htmlFor,
  className,
  children,
}: {
  label?: string
  htmlFor?: string
  className?: string
  children: ReactNode
}) {
  const labelClass =
    'w-[var(--ui-label-width,2.25rem)] shrink-0 text-[13px] text-[color:var(--ui-label)]'
  return (
    <div
      className={cn(
        'flex h-[var(--ui-row-height,3rem)] min-w-0 items-center gap-3 px-3.5',
        className,
      )}
    >
      {label &&
        (htmlFor ? (
          <label htmlFor={htmlFor} className={labelClass}>
            {label}
          </label>
        ) : (
          <span className={labelClass}>{label}</span>
        ))}
      <div className="flex min-w-0 flex-1 items-center gap-2">{children}</div>
    </div>
  )
}

/** A raised pop-up button with a native-style up/down chevron. */
export function PopupSelect<T extends Option>({
  label,
  value,
  items,
  disabled,
  onChange,
  onOpenChange,
  renderIcon,
}: {
  label: string
  value: string
  items: readonly T[]
  disabled?: boolean
  onChange: (value: string) => void
  onOpenChange?: (open: boolean) => void
  renderIcon?: (item: T) => ReactNode
}) {
  const selected = items.find((item) => item.value === value)
  return (
    <Select
      items={items}
      value={value}
      disabled={disabled}
      onOpenChange={onOpenChange}
      onValueChange={(next) => next !== null && onChange(next)}
    >
      <SelectTrigger
        className={cn(
          'h-[28px]! w-full min-w-0 gap-2 rounded-[7px] border border-[var(--ui-control-border)] py-0 pr-1.5 pl-2.5',
          'bg-[var(--ui-control)]! text-[13px] font-normal text-[color:var(--ui-label)] shadow-[var(--ui-control-shadow)]',
          'transition-[background-color] duration-150 hover:bg-[var(--ui-control-hover)]!',
          '[&>svg:last-child]:hidden',
          "[&_[data-slot='select-value']]:block [&_[data-slot='select-value']]:min-w-0",
          "[&_[data-slot='select-value']]:truncate",
        )}
        aria-label={label}
        title={selected?.label}
      >
        {selected && renderIcon?.(selected)}
        <SelectValue />
        <ChevronsUpDown
          className="size-3! shrink-0 text-[color:var(--ui-secondary)]"
          strokeWidth={2}
          aria-hidden="true"
        />
      </SelectTrigger>
      <SelectContent
        className="w-[max(var(--anchor-width),_200px)] max-w-[min(380px,_calc(100vw_-_24px))] rounded-[10px] p-1 text-[13px] [animation-duration:var(--motion-duration)] [animation-timing-function:var(--motion-ease)] data-ending-style:[animation-duration:var(--motion-close-duration)] [&_[data-slot='select-item']]:min-h-[28px] [&_[data-slot='select-item']]:rounded-[6px]"
        align="start"
        sideOffset={6}
        alignItemWithTrigger={false}
      >
        {items.map((item) => (
          <SelectItem
            key={item.value}
            value={item.value}
            title={item.label}
            className="[&>:first-child]:min-w-0 [&>:first-child]:shrink"
          >
            {renderIcon?.(item)}
            <span className="min-w-0 truncate">{item.label}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

/** A recessed track with a raised thumb that slides to the checked option. */
export function SegmentedControl<T extends string>({
  name,
  label,
  value,
  options,
  onChange,
  className,
}: {
  name: string
  label: string
  value: T
  options: readonly { value: T; label: string }[]
  onChange: (value: T) => void
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex h-[26px] flex-1 rounded-[8px] bg-[var(--ui-track)] p-[2px] shadow-[var(--ui-track-inset)]',
        className,
      )}
      role="radiogroup"
      aria-label={label}
    >
      {options.map((option) => {
        const checked = value === option.value
        return (
          <label
            key={option.value}
            className="relative flex-1 cursor-pointer has-focus-visible:rounded-[6px] has-focus-visible:outline-2 has-focus-visible:outline-[var(--color-ring)]"
          >
            <input
              className="sr-only"
              type="radio"
              name={name}
              value={option.value}
              checked={checked}
              onChange={() => onChange(option.value)}
            />
            {checked && (
              <motion.span
                layoutId={`${name}-thumb`}
                className="absolute inset-0 rounded-[6px] border border-[var(--ui-control-border)] bg-[var(--ui-control)] shadow-[var(--ui-control-shadow)]"
                transition={{ type: 'spring', stiffness: 520, damping: 40 }}
              />
            )}
            <span
              className={cn(
                'relative grid h-full place-items-center text-[12px] transition-colors duration-150',
                checked
                  ? 'font-medium text-[color:var(--ui-label)]'
                  : 'text-[color:var(--ui-secondary)] hover:text-[color:var(--ui-label)]',
              )}
            >
              {option.label}
            </span>
          </label>
        )
      })}
    </div>
  )
}
