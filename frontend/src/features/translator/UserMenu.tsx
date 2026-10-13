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
          <Button variant="ghost" size="icon" className="user-menu-trigger" />
        }
        aria-label="用户菜单"
        title="用户菜单"
      >
        <Avatar.Root className="user-avatar">
          <Avatar.Fallback>{previewAccount.initial}</Avatar.Fallback>
        </Avatar.Root>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        className="user-menu-content"
        align="end"
        alignOffset={-16}
        sideOffset={4}
        aria-label="用户菜单"
      >
        <Menu.Arrow className="user-menu-arrow">
          <svg width="14" height="8" viewBox="0 0 14 8" fill="none">
            <path d="M0 8L6.3 1.7a1 1 0 0 1 1.4 0L14 8" />
          </svg>
        </Menu.Arrow>
        <DropdownMenuGroup className="user-menu-account">
          <DropdownMenuLabel className="user-menu-balance">
            <span>额度</span>
            <strong>
              {previewAccount.points.toLocaleString('en-US')} 积分
            </strong>
          </DropdownMenuLabel>
          <DropdownMenuSeparator className="user-menu-separator" />
          <DropdownMenuItem
            className="user-menu-item"
            onClick={() => placeholder('充值额度')}
          >
            充值额度
          </DropdownMenuItem>
          <DropdownMenuItem
            className="user-menu-item"
            onClick={() => placeholder('积分明细')}
          >
            积分明细
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="user-menu-separator" />
        <DropdownMenuGroup>
          <DropdownMenuItem
            className="user-menu-item"
            onClick={() => placeholder('操作手册')}
          >
            操作手册
          </DropdownMenuItem>
          <DropdownMenuItem
            className="user-menu-item"
            onClick={() => placeholder('设置')}
          >
            设置
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="user-menu-separator" />
        <div className="user-menu-appearance">
          <span id="user-menu-appearance-label">外观</span>
          <DropdownMenuRadioGroup
            className="appearance-options"
            aria-labelledby="user-menu-appearance-label"
            value={appearance}
            onValueChange={(value) => {
              if (isAppearance(value)) setAppearance(value)
            }}
          >
            {appearances.map(({ value, label, Icon }) => (
              <DropdownMenuRadioItem
                key={value}
                className="appearance-option"
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
