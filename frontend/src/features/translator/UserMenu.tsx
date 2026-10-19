import { cn } from '@/lib/utils'
import { Avatar } from '@base-ui/react/avatar'
import { Menu } from '@base-ui/react/menu'
import { Contrast, Moon, Sun } from 'lucide-react'
import { isAppearance, useAppearance } from '@/app/useAppearance'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useToastManager } from '@/components/ui/toast'

const appearances = [
  { value: 'light', label: '浅色', Icon: Sun },
  { value: 'system', label: '跟随系统', Icon: Contrast },
  { value: 'dark', label: '深色', Icon: Moon },
] as const

// shortcut: Screenshot account data; replace with the account API when available.
const previewAccount = { initial: 'Y', points: 14_320 }

export function UserMenu() {
  const { appearance, setAppearance } = useAppearance()
  const toastManager = useToastManager()
  const placeholder = (label: string) =>
    toastManager.add({
      id: 'user-menu-notice',
      title: `${label}功能暂未开放`,
    })

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="absolute [top:37px] [right:23px] w-[40px] h-[40px] p-0 border-0 rounded-[50%] bg-transparent [--wails-draggable:no-drag]"
          />
        }
        aria-label="用户菜单"
        title="用户菜单"
      >
        <Avatar.Root className="grid place-items-center w-[40px] h-[40px] rounded-[50%] bg-[var(--user-avatar-background)] text-[color:var(--user-avatar-foreground)] text-[13px] font-normal">
          <Avatar.Fallback>{previewAccount.initial}</Avatar.Fallback>
        </Avatar.Root>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        className={cn(
          "[&[data-slot='dropdown-menu-content']]:w-[216px] [&[data-slot='dropdown-menu-content']]:max-w-[calc(100vw_-_24px)]",
          "[&[data-slot='dropdown-menu-content']]:pt-0 [&[data-slot='dropdown-menu-content']]:px-4.5",
          "[&[data-slot='dropdown-menu-content']]:pb-2 [&[data-slot='dropdown-menu-content']]:overflow-visible",
          "[&[data-slot='dropdown-menu-content']]:border [&[data-slot='dropdown-menu-content']]:border-[var(--user-menu-border)]",
          "[&[data-slot='dropdown-menu-content']]:rounded-[8px] [&[data-slot='dropdown-menu-content']]:bg-popover",
          "[&[data-slot='dropdown-menu-content']]:text-[color:var(--user-menu-foreground)]",
          "[&[data-slot='dropdown-menu-content']]:shadow-[0_10px_28px_#20244c12]",
          "[&[data-slot='dropdown-menu-content']]:[transition-property:opacity,_transform]",
          "[&[data-slot='dropdown-menu-content']]:[--wails-draggable:no-drag] [&_[data-slot='dropdown-menu-group']]:py-[7px]",
          "[&_[data-slot='dropdown-menu-group']]:px-0 [&_.user-menu-account]:pt-0!",
        )}
        align="end"
        alignOffset={-16}
        sideOffset={4}
        aria-label="用户菜单"
      >
        <Menu.Arrow className="flex [&[data-side='bottom']]:[top:-8px] [&[data-side='top']]:[bottom:-8px] [&[data-side='top']]:[transform:rotate(180deg)] [&_svg]:[fill:var(--color-popover)] [&_svg]:[stroke:var(--user-menu-border)]">
          <svg width="14" height="8" viewBox="0 0 14 8" fill="none">
            <path d="M0 8L6.3 1.7a1 1 0 0 1 1.4 0L14 8" />
          </svg>
        </Menu.Arrow>
        <DropdownMenuGroup className="user-menu-account [&_.user-menu-separator]:mb-2">
          <DropdownMenuLabel className="flex items-center gap-3 h-[52px] py-0 px-0.5 text-[13px] leading-[1.3333333333] font-normal text-[color:var(--user-menu-muted)] [&_strong]:text-[13px] [&_strong]:font-bold [&_strong]:text-[color:var(--user-menu-foreground)]">
            <span>额度</span>
            <strong>
              {previewAccount.points.toLocaleString('en-US')} 积分
            </strong>
          </DropdownMenuLabel>
          <DropdownMenuSeparator className="user-menu-separator m-0 bg-[var(--user-menu-border)]" />
          <DropdownMenuItem
            className="h-[34px] my-0 mx-[-8px] py-0 px-2.5 text-[13px] font-normal cursor-pointer [&[data-highlighted]]:bg-[var(--user-menu-hover)] [&[data-highlighted]]:text-[color:var(--user-menu-foreground)] focus:bg-[var(--user-menu-hover)] focus:text-[color:var(--user-menu-foreground)]"
            onClick={() => placeholder('充值额度')}
          >
            充值额度
          </DropdownMenuItem>
          <DropdownMenuItem
            className="h-[34px] my-0 mx-[-8px] py-0 px-2.5 text-[13px] font-normal cursor-pointer [&[data-highlighted]]:bg-[var(--user-menu-hover)] [&[data-highlighted]]:text-[color:var(--user-menu-foreground)] focus:bg-[var(--user-menu-hover)] focus:text-[color:var(--user-menu-foreground)]"
            onClick={() => placeholder('积分明细')}
          >
            积分明细
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="user-menu-separator m-0 bg-[var(--user-menu-border)]" />
        <DropdownMenuGroup>
          <DropdownMenuItem
            className="h-[34px] my-0 mx-[-8px] py-0 px-2.5 text-[13px] font-normal cursor-pointer [&[data-highlighted]]:bg-[var(--user-menu-hover)] [&[data-highlighted]]:text-[color:var(--user-menu-foreground)] focus:bg-[var(--user-menu-hover)] focus:text-[color:var(--user-menu-foreground)]"
            onClick={() => placeholder('操作手册')}
          >
            操作手册
          </DropdownMenuItem>
          <DropdownMenuItem
            className="h-[34px] my-0 mx-[-8px] py-0 px-2.5 text-[13px] font-normal cursor-pointer [&[data-highlighted]]:bg-[var(--user-menu-hover)] [&[data-highlighted]]:text-[color:var(--user-menu-foreground)] focus:bg-[var(--user-menu-hover)] focus:text-[color:var(--user-menu-foreground)]"
            onClick={() => placeholder('设置')}
          >
            设置
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="user-menu-separator m-0 bg-[var(--user-menu-border)]" />
        <div className="flex items-center justify-between gap-4 pt-[9px] text-[13px] text-[color:var(--user-menu-foreground)]">
          <span id="user-menu-appearance-label">外观</span>
          <DropdownMenuRadioGroup
            className="flex h-[32px] p-0 border border-[var(--user-menu-border)] rounded-[10px]"
            aria-labelledby="user-menu-appearance-label"
            value={appearance}
            onValueChange={(value) => {
              if (isAppearance(value)) setAppearance(value)
            }}
          >
            {appearances.map(({ value, label, Icon }) => (
              <DropdownMenuRadioItem
                key={value}
                className={cn(
                  'grid place-items-center w-[36px] p-0 rounded-[9px] text-[color:var(--user-menu-foreground)] cursor-pointer [&_svg]:w-[18px]!',
                  '[&_svg]:h-[18px]! [&_.lucide-contrast_path]:[fill:currentColor] [&_.lucide-contrast_path]:[transform:scale(1.6)]',
                  "[&_.lucide-contrast_path]:[transform-origin:12px_12px] [&_[data-slot='dropdown-menu-radio-item-indicator']]:hidden",
                  '[&[data-checked]]:bg-[var(--user-menu-selected)] [&[data-highlighted]]:bg-[var(--user-menu-hover)]',
                  '[&[data-highlighted]]:text-[color:var(--user-menu-foreground)] focus:bg-[var(--user-menu-hover)]',
                  'focus:text-[color:var(--user-menu-foreground)] focus-visible:[outline:2px_solid_var(--user-menu-foreground)]',
                  'focus-visible:[outline-offset:-2px]',
                )}
                value={value}
                aria-label={label}
                label={label}
                title={label}
                closeOnClick={false}
              >
                <Icon aria-hidden="true" strokeWidth={1.5} />
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
