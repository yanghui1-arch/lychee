# 荔枝 · Lychee

基于 Wails v2 的桌面翻译客户端，前端使用 React、TypeScript、Tailwind CSS 和 Vite。
当前版本采用白色双栏工作区，顶部导航为「文译 / 同传 / 字幕 / 工具」。使用 #E8E8E8 细边框、12–14px 圆角、Inter 字体和黑色强调。输入区只保留面板外框，textarea 与其容器没有内层边框。
界面不包含品牌区、Apple 三色按钮、清空全文、术语库、场景选择、语音、导入文档或翻译操作按钮。

## 当前功能

- 文译 / 同传 / 字幕 / 工具导航；文译提供双栏编辑区与原文、译文语言选择。
- 右上角用户菜单复刻参考图中的头像、额度、充值额度、积分明细、操作手册、设置和外观选项。浅色 / 跟随系统 / 深色会实际切换全局主题，选择保存在本机 `lychee.appearance.v1`；默认跟随系统。头像 Y 和 14,320 积分为截图展示数据，四个业务入口暂作占位，点击提示尚未开放，待接入账户服务。
- 文译两栏始终等高，默认占内容区一半高度；随原文增长同步增高，最多占内容区 90%，超出后在框内滚动，删短文字后自动缩回。
- 5,000 字输入限制、动态字数、复制与译文喜欢 / 取消喜欢；两栏间的细线双向箭头可交换原文与译文语言。
- 字幕页提供双语示例预览、语言选择、带图标的监听来源、五种字幕样式和字号、顶部 / 底部 / 自定义位置设置；自定义位置可拖动或用方向键微调。工具页目前显示内容占位；截图翻译、图片翻译等功能归入工具页。
- 原文停止输入 600 毫秒后自动翻译；中文输入法组词期间不请求。切换标签会保留已发出的请求和译文，返回文译页不会重复请求；尚未发出的请求在返回后重新计时。修改内容、切换语言或点击重试会取消旧请求，旧译文不会覆盖新内容。
- 状态在本机持久化，字体随应用打包；窗口固定为 860 × 600 逻辑像素，禁用调整大小与最大化。Windows / Linux 在右上角显示最小化、关闭线条按钮，macOS 在左上角显示关闭、最小化圆点按钮；点击调用 Wails 窗口接口，标题栏空白区域可拖动窗口。
- 翻译页调用真实服务，支持等待、加载、错误和重试，复制按钮复制当前译文。原语言支持自动识别；同传尚未接入语音服务。

## 字幕音频监听

Windows 64 位桌面端在「字幕 → 监听来源」默认选择「全局监听」，并列出已打开桌面窗口的应用；不提供麦克风监听。旧版设置首次迁移时将来源设为全局监听，保留语言与字幕外观设置，后续记住用户选择。不要求先播放声音或建立音频会话。同时补充正在播放声音的后台应用，不使用已停止播放的残留音频会话判定应用是否打开。界面可见时每 2 秒刷新列表，恢复窗口或打开下拉框时也会刷新。选择应用后打开桌面字幕开关，界面会显示监听来源、等待 / 检测到声音和实时音量条。会议结束后应用主窗口仍打开时继续列出该应用，静音时显示等待声音。监听中切换来源会先停止旧采集；切换工作区标签保持监听，关闭开关或退出客户端释放资源。目标进程退出后停止监听并提示，应用重启后需要重新选择。

应用选项与选中项动态显示该进程 EXE 中的图标，无需预置图标素材。图标转换为透明的 32×32 PNG，通过本机 Wails 桥接传到界面；内存缓存最多 128 个可执行文件，文件修改后重新提取，读取或加载失败时显示通用应用图标。

同一应用进程及子进程只列出一项。多个独立的同名应用保留各自的监听目标，以可见窗口标题区分；标题相同或没有标题时使用实例编号。界面不显示 PID；长标题省略显示，悬停可查看完整名称。浏览器按进程树监听，当前不提供标签页选择。

应用列表过滤 Windows 窗口宿主、文字输入、桌面外壳、搜索等系统辅助进程，以及 Realtek 音频控制台；Windows `SystemApps` 与系统设置目录中的组件也不列出。窗口枚举跳过 DWM 隐藏的窗口。过滤同样应用于后台音频会话，避免系统组件播放声音后重新出现；保留普通桌面软件和商店安装的用户应用。

单独应用使用 Windows [进程回环采集](https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/)，包含目标进程及子进程，不采集其他应用的声音；需要 Windows 11 或 Windows build 20348 及以上。按相同可执行文件的父子进程合并浏览器等多进程应用，并用进程创建时间防止重启后的 PID 复用。全局监听使用默认输出设备的 [WASAPI loopback](https://learn.microsoft.com/en-us/windows/win32/coreaudio/loopback-recording) 采集混合声音，设备断开会显示错误。

开启字幕并成功连接音频来源后，桌面会显示独立的置顶字幕框。主窗口最小化或切换工作区时仍然显示；字幕框不抢焦点、穿透鼠标操作，不出现在任务栏。语言、字号、五种样式和顶部／底部／自定义位置与设置页同步，自定义位置通过预览拖动或方向键调整。字幕显示在开启时主窗口所在显示器的工作区内，适配 DPI 缩放并避开任务栏；关闭开关、音频采集失败或退出应用时收起并释放资源。

桌面字幕当前使用与预览一致的双语占位文字。`subtitlePlaceholderText` 中以 Go 风格的函数注释与 TODO 标明后续语音识别和翻译接入点，尚未输出真实转写内容。

默认「深底白字」按参考图使用约 70% 不透明度的黑色底（约 30% 透出画面），保证白字可读性。其他预设保留轻量选择：暖黄描边为完全无底，轻雾白字、浅底深字、暗夜轻底分别使用约 10%、15%、20% 不透明度的背景。文字保持实色，并绘制细描边与轻阴影；取消背景模糊、边框和圆角。预览与桌面字幕的西文统一使用内置 Roboto Medium（500 字重）；桌面端仅在应用进程内加载，不修改系统字体。中文 Microsoft YaHei UI、日文 Yu Gothic UI、韩文 Malgun Gothic 使用系统字体，缺失时由系统回退。原生文字以两倍分辨率绘制后缩小，改善透明窗口中的抗锯齿；默认深底款使用更轻的描边。

采集输出为 16 kHz、单声道、16 位 PCM，每 200 毫秒通过 Wails `audio:chunk` 事件提供一块数据（`pcm` 为 Base64，附带 `session`、`sampleRate`、`channels` 和 `level`）。`audio:status` 提供生命周期状态。音频只在内存中处理，不写入文件或上传；目前尚未接入语音识别或音频翻译。

普通测试运行 `go test ./...`。原生隔离测试会短暂播放两种低音量测试音，并验证只收到目标应用的测试音、应用退出后停止采集：

```powershell
$env:LYCHEE_AUDIO_HARDWARE_TEST = '1'
go test -v -run TestApplicationAudioHardware -count=1 .
Remove-Item Env:LYCHEE_AUDIO_HARDWARE_TEST
```

## 翻译服务配置

默认服务地址为 `http://127.0.0.1:8000/v1/translate`，使用 POST，发送 `text`、`target_language` 和 `source_language`（默认 `auto`），读取响应的 `translated_text`。界面语言使用标准语言代码，例如 `en-US`、`zh-CN`。

开发时或运行客户端前，可通过环境变量覆盖**完整地址，包含路由**：

```powershell
$env:LYCHEE_TRANSLATE_URL = 'http://127.0.0.1:8000/v1/translate'
wails dev
# 仅预览前端时也使用同一环境变量：
bun run --cwd frontend dev
```

发布时把远程默认地址写入客户端，无需用户设置环境变量：

```powershell
wails build -ldflags '-X main.translationEndpoint=https://api.example.com/v1/translate'
```

`LYCHEE_TRANSLATE_URL` 的运行时配置优先于打包默认值。地址或路由变更只需修改该配置。桌面端由 Wails 的 Go HTTP 代理发送请求，浏览器开发预览由 Vite 代理发送请求，两者均通过同源 `/api/translate` 访问，避免服务端 CORS 配置差异。请求超时为 30 秒。

## 开发

需要 Go 1.25+、Bun、Wails CLI 2.15.0，以及 Windows WebView2 Runtime。

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
bun install --cwd frontend --frozen-lockfile
go mod download
wails dev
```

仅预览前端：

```sh
bun run --cwd frontend dev
```

检查与打包：

```sh
bun run --cwd frontend typecheck
bun run --cwd frontend format:check
bun test --cwd frontend tests
go test -race ./...
go vet ./...
wails build
```

Windows 应用输出为 `build/bin/lychee.exe`。`build/` 中的平台配置属于源码，
`build/bin/`、`frontend/dist/` 和 `node_modules/` 是忽略的构建输出。

## 项目结构

```text
main.go                         Wails 窗口配置与前端资源嵌入
translation.go                  可配置的翻译服务 HTTP 代理
build/                          平台资源和打包配置
frontend/
  src/
    app/                        应用入口与全局行为
    components/ui/              上游 UI 组件与适配
    features/translator/        工作区、输入检测、翻译请求与状态
    lib/                        桌面桥接与通用工具
    styles/                     设计变量、布局和动效
  wailsjs/runtime/               Wails 生成的运行时桥接
  licenses/                     上游组件与字体授权文本
```

工作区状态集中在 `useTranslator.ts`，存储于本机 `lychee.workspace.v1`。
历史记录最多 20 条，收藏最多 50 条；浏览器预览和桌面运行使用各自的本机存储。
页面控件直接组合 shadcn 组件，功能目录只承担布局和业务状态。
翻译接口集中在 `translation-api.ts`，输入防抖、输入法与请求取消集中在 `useTranslation.ts`；工作区组件只组合控件与显示状态。

## 组件来源

- [Transitions.dev](https://transitions.dev/library.html)：顶部导航复用 Tabs sliding，译文操作复用 Like button 的心形填充、弹跳与八颗粒子动效，保留其默认粉红色选中效果。复制与保存提示使用 Toast open / close 的上浮、淡入、模糊和缩放。状态和键盘操作使用 shadcn/Base UI；菜单使用其动效参数。
- [Rare UI](https://www.rareui.com/components)：Animated Counter，保留原始实现和署名。
- [shadcn/ui](https://ui.shadcn.com)：Base Nova Tabs、Table、Input、Button、Toggle、Select、Dropdown Menu 与 Textarea，底层使用 Base UI。Tabs 的方向样式适配当前 Base UI 的 `data-orientation` 属性。
- [Base UI Toast](https://base-ui.com/react/components/toast)：管理提示的生命周期、自动关闭、重复消息与无障碍播报。
- [Base UI Avatar](https://base-ui.com/react/components/avatar)：用户头像回退字母；用户菜单沿用 shadcn Dropdown Menu，外观选择使用其 Radio Group / Radio Item，弹出箭头由 Base UI Menu Arrow 定位。
- [Fontsource](https://fontsource.org/fonts/inter)：统一使用自托管 Inter 可变字体，OFL 授权随源码保留；Inter 未覆盖的中文字形由系统无衬线字体补齐。
- [Roboto](https://github.com/googlefonts/roboto-3-classic/releases/tag/v3.015)：字幕使用自带的 Roboto Medium 3.015，预览与 Windows 原生窗口共用同一份 TTF。OFL 授权与下载来源记录保存在 `frontend/licenses/roboto-OFL.txt` 和 `roboto-SOURCE.txt`。

授权原文位于 `frontend/licenses/`，本 README 保留组件来源链接与署名。
Rare UI 使用 MIT + Commons Clause + Attribution 条款，具体授权范围以该目录中的原文为准。
