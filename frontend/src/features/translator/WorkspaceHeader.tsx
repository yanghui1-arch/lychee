import { useEffect, useState } from 'react'
import { Minus, X } from 'lucide-react'
import lycheeLogo from '@/assets/lycheeLogo.webp'
import { Button } from '@/components/ui/button'
import { TabsIndicator, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  browserPlatform,
  closeApp,
  getPlatform,
  minimizeApp,
} from '@/lib/desktop'

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
    <header className="workspace-header">
      <TabsList className="workspace-navigation t-tabs" aria-label="工作区导航">
        <img
          className="workspace-logo"
          src={lycheeLogo}
          alt="荔枝翻译"
          width={28}
          height={28}
          draggable={false}
        />
        <TabsTrigger className="t-tab" value="translate">
          翻译
        </TabsTrigger>
        <TabsTrigger className="t-tab" value="interpret">
          同传
        </TabsTrigger>
        <TabsTrigger className="t-tab" value="settings">
          设置
        </TabsTrigger>
        <TabsTrigger className="t-tab" value="manual">
          手册
        </TabsTrigger>
        <TabsIndicator className="t-tabs-pill" aria-hidden="true" />
      </TabsList>
      <div
        className="window-controls"
        data-platform={platform}
        role="group"
        aria-label="窗口操作"
      >
        {controls.map(({ action, label, Icon, onClick }) => (
          <Button
            key={action}
            className="window-control"
            data-action={action}
            variant="ghost"
            size="icon"
            aria-label={label}
            title={label}
            onClick={onClick}
          >
            <span className="window-control-symbol">
              <Icon className="size-3.5" strokeWidth={1.5} />
            </span>
          </Button>
        ))}
      </div>
    </header>
  )
}
