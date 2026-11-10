import { cn } from '@/lib/utils'
import { useEffect, useState } from 'react'
import { Minus, X, Scissors } from 'lucide-react'
import { BeginScreenshot } from '../../../wailsjs/go/app/WindowService'
import lycheeLogo from '@/assets/lycheeLogo.webp'
import { Button } from '@/components/ui/button'
import { TabsIndicator, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { UserMenu } from './UserMenu'
import {
  browserPlatform,
  closeApp,
  getPlatform,
  minimizeApp,
} from '@/lib/desktop'

// Sliding tab timing: Transitions.dev, Jakub Antalik.
// License: frontend/licenses/transitions-LICENSE.txt

export function WorkspaceHeader() {
  const [platform, setPlatform] = useState(browserPlatform)
  useEffect(() => {
    void getPlatform().then(setPlatform).catch(console.error)
  }, [])
  const controls = [
    { action: 'close', label: '关闭荔枝', Icon: X, onClick: closeApp },
    {
      action: 'minimize',
      label: '最小化窗口',
      Icon: Minus,
      onClick: minimizeApp,
    },
  ]
  if (platform !== 'darwin') controls.reverse()

  return (
    <header className="h-[84px] pt-2 shrink-0 relative flex items-start justify-end [--wails-draggable:drag]">
      <TabsList
        className={cn(
          "[&_[data-slot='tabs-trigger']:not([aria-selected='true'])]:text-[color:var(--tabs-text-muted)] [--tabs-dur:250ms]",
          '[--tabs-ease:cubic-bezier(0.22,1,0.36,1)] inline-flex items-center gap-[3px] bg-[var(--tabs-bar-bg)]',
          '[--tabs-bar-bg:var(--color-background)] [--tabs-pill-bg:var(--color-primary)] [--tabs-text-muted:var(--subtle-foreground)]',
          '[--tabs-text-active:var(--color-primary)] absolute [top:40px] [left:50%] [transform:translateX(-50%)] w-[324px] h-[34px]!',
          "p-0.5 border border-border rounded-[14px] [--wails-draggable:no-drag] [&_[data-slot='tabs-trigger']]:h-[28px]",
          "[&_[data-slot='tabs-trigger']]:rounded-[12px] [&_[data-slot='tabs-trigger']]:text-[12px]",
          "[&_[data-slot='tabs-trigger']]:font-medium [&_[data-slot='tabs-trigger']]:bg-transparent",
          "[&_[data-slot='tabs-trigger']]:shadow-none! [&_[aria-selected='true']]:text-primary-foreground!",
          '[&_.t-tabs-pill]:rounded-[12px]!',
        )}
        aria-label="工作区导航"
      >
        <img
          className="[flex:0_0_28px] w-[28px] h-[28px] mr-3 object-contain"
          src={lycheeLogo}
          alt="荔枝翻译"
          width={28}
          height={28}
          draggable={false}
        />
        <TabsTrigger
          className={cn(
            'relative appearance-none border-0 bg-transparent h-[30px] py-1 px-3 text-[color:var(--tabs-text-muted)] cursor-pointer',
            'rounded-[48px] z-[1] [transition:color_var(--tabs-dur)_var(--tabs-ease)]',
            "[&:not([aria-selected='true']):hover]:text-[color:var(--tabs-text-active)]",
            "[&[aria-selected='true']]:text-[color:var(--tabs-text-active)] [@media(prefers-reduced-motion:reduce)]:[transition:none]!",
          )}
          value="translate"
        >
          文译
        </TabsTrigger>
        <TabsTrigger
          className={cn(
            'relative appearance-none border-0 bg-transparent h-[30px] py-1 px-3 text-[color:var(--tabs-text-muted)] cursor-pointer',
            'rounded-[48px] z-[1] [transition:color_var(--tabs-dur)_var(--tabs-ease)]',
            "[&:not([aria-selected='true']):hover]:text-[color:var(--tabs-text-active)]",
            "[&[aria-selected='true']]:text-[color:var(--tabs-text-active)] [@media(prefers-reduced-motion:reduce)]:[transition:none]!",
          )}
          value="interpret"
        >
          同传
        </TabsTrigger>
        <TabsTrigger
          className={cn(
            'relative appearance-none border-0 bg-transparent h-[30px] py-1 px-3 text-[color:var(--tabs-text-muted)] cursor-pointer',
            'rounded-[48px] z-[1] [transition:color_var(--tabs-dur)_var(--tabs-ease)]',
            "[&:not([aria-selected='true']):hover]:text-[color:var(--tabs-text-active)]",
            "[&[aria-selected='true']]:text-[color:var(--tabs-text-active)] [@media(prefers-reduced-motion:reduce)]:[transition:none]!",
          )}
          value="subtitle"
        >
          字幕
        </TabsTrigger>
        <TabsIndicator
          className={cn(
            't-tabs-pill absolute [top:var(--active-tab-top)] [left:0] h-[var(--active-tab-height)] w-[var(--active-tab-width)]',
            'bg-[var(--tabs-pill-bg)] rounded-[48px] [transform:translateX(var(--active-tab-left))]',
            '[transition:transform_var(--tabs-dur)_var(--tabs-ease),_width_var(--tabs-dur)_var(--tabs-ease)] [will-change:transform,_width]',
            'z-[0] pointer-events-none [@media(prefers-reduced-motion:reduce)]:[transition:none]!',
          )}
          aria-hidden="true"
        />
      </TabsList>
      <UserMenu />
      <Button
        className="absolute top-2 right-20 [--wails-draggable:no-drag]"
        variant="ghost"
        size="icon"
        aria-label="截图"
        title="截图 · F1"
        onClick={() => void BeginScreenshot().catch(console.error)}
      >
        <Scissors size={16} />
      </Button>
      <div
        className={cn(
          "flex items-center gap-1 [--wails-draggable:no-drag] [&[data-platform='darwin']]:mr-auto [&[data-platform='darwin']]:gap-0",
          "[&[data-platform='darwin']_.window-control]:w-[24px] [&[data-platform='darwin']_.window-control]:h-[24px]",
          "[&[data-platform='darwin']_.window-control]:bg-transparent [&[data-platform='darwin']_.window-control-symbol]:w-[12px]",
          "[&[data-platform='darwin']_.window-control-symbol]:h-[12px] [&[data-platform='darwin']_.window-control-symbol]:rounded-[50%]",
          "[&[data-platform='darwin']_.window-control-symbol]:bg-[#febc2e]",
          "[&[data-platform='darwin']_.window-control-symbol]:shadow-[inset_0_0_0_0.5px_#0000001a]",
          "[&[data-platform='darwin']_[data-action='close']_.window-control-symbol]:bg-[#ff5f57] [&[data-platform='darwin']_svg]:w-[8px]",
          "[&[data-platform='darwin']_svg]:h-[8px] [&[data-platform='darwin']_svg]:text-[color:#00000099]",
          "[&[data-platform='darwin']_svg]:[stroke-width:2] [&[data-platform='darwin']_svg]:opacity-0",
          "[&[data-platform='darwin']_svg]:[transition:opacity_120ms_ease] [&[data-platform='darwin']:hover_svg]:opacity-100",
          "[&[data-platform='darwin']_:focus-visible_svg]:opacity-100",
        )}
        data-platform={platform}
        role="group"
        aria-label="窗口操作"
      >
        {controls.map(({ action, label, Icon, onClick }) => (
          <Button
            key={action}
            className="window-control w-[32px] h-[32px] text-[color:var(--subtle-foreground)] cursor-default [&[data-action='close']:hover]:bg-[#e81123] [&[data-action='close']:hover]:text-white"
            data-action={action}
            variant="ghost"
            size="icon"
            aria-label={label}
            title={label}
            onClick={onClick}
          >
            <span className="window-control-symbol grid place-items-center">
              <Icon className="size-3.5" strokeWidth={1.5} />
            </span>
          </Button>
        ))}
      </div>
    </header>
  )
}
