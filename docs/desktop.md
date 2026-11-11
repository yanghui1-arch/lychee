# 悬浮球与窗口

悬浮球是桌面常驻入口，负责打开工作区和截图。工作区独立显示文译、同传及字幕页面；托盘负责恢复窗口和完整退出。

## 启动与独立工作区

启动先显示置顶、透明背景的悬浮球，球体为 56 × 56 逻辑像素。双击球体或聚焦后按 Enter / 空格打开 860 × 600 的工作区，悬浮球仍保持可用。工作区固定大小，不支持用户调整大小或最大化。

Wails v2 每个进程使用一个 WebView 窗口，因此工作区由同一个 exe 启动独立子进程，内部通过 `LYCHEE_WORKSPACE_PARENT` 标识所属主进程。后续打开操作复用该工作区并将其激活，不反复创建窗口。子进程监视父进程句柄，父进程退出后自行结束；该内部环境变量不是用户配置项。

主应用具有单实例入口，重复启动 exe 会唤起已有应用；为实现独立工作区启动的受控子进程属于正常设计。

实现：[app.go](../internal/app.go)、[window.go](../internal/window.go)、[workspace_windows.go](../internal/workspace_windows.go)、[App.tsx](../frontend/src/app/App.tsx)、[FloatingBall.tsx](../frontend/src/app/FloatingBall.tsx)。

## 拖动与位置恢复

按住悬浮球可拖动。首次启动或保存坐标超出当前屏幕时，将球心放在主屏宽度约 75%、高度约 25% 处。恢复坐标时结合物理像素和逻辑尺寸检查可见球体，允许透明工具区边距越过屏幕。

位置写入用户配置目录的 `lychee/floating-position.json`，内容是 `[x,y]`。先写临时文件再替换旧文件，避免写失败破坏上次位置。截图、尚未显示、工作区或最小化状态下不会覆盖球的位置。工作区移动不改变悬浮球位置。

实现：[window_position.go](../internal/window_position.go)、[window_windows.go](../internal/window_windows.go)。

## 悬停工具栏与点击区域

鼠标进入时，工具栏从球体向左展开，提供截图按钮，右上方显示关闭图标；工具栏跟随主题。GSAP 时间轴控制抽屉和叉号，快速移入移出时从当前进度反向播放，卸载时清理动画。

窗口固定使用 118 × 66 的透明画布，悬停不移动、不缩放 WebView。后端只调整原生窗口区域：展开时接收工具区输入，收回动画结束后释放透明区域，避免挡住桌面点击。递增 revision 拒绝过期回调，防止旧的收回状态覆盖新悬停。

离开 3 秒后球体变暗并降至 45% 不透明度，颜色与透明度在 500ms 内过渡；重新进入立即取消变暗计时并恢复亮态。截图事件暂时切换主窗口为截图编辑器，结束后恢复。

实现：[FloatingBall.tsx](../frontend/src/app/FloatingBall.tsx)、[useFloatingControlsAnimation.ts](../frontend/src/app/useFloatingControlsAnimation.ts)、[window.go](../internal/window.go)、[window_windows.go](../internal/window_windows.go)。截图操作见[截图文档](screenshot.md)。

## 关闭、最小化与托盘

工作区的关闭按钮隐藏工作区，双击悬浮球可以重新显示；悬浮球的关闭入口会保存位置并最小化悬浮窗口；Alt+F4 等关闭请求也由窗口服务拦截，按所在窗口执行隐藏或最小化。后台监听不因隐藏工作区而停止。只显示悬浮球时不占用任务栏，工作区显示时有任务栏入口。

点击托盘或使用“显示荔枝”恢复应用，选择“退出荔枝”才执行完整退出，释放监听、字幕和窗口资源，并使工作区子进程结束。截图及桌面贴图的取消行为由截图服务管理。

实现：[close_behavior.go](../internal/close_behavior.go)、[tray_windows.go](../internal/tray_windows.go)、[window_windows.go](../internal/window_windows.go)、[desktop.ts](../frontend/src/lib/desktop.ts)。

## 验证

窗口测试位于 `internal/window*_test.go`。桌面手测应覆盖首次启动位置、移动后重启、重复启动、反复打开工作区、快速悬停、截图恢复、隐藏期间继续监听及托盘退出。
