# 荔枝 · Lychee

基于 Wails v2 的桌面翻译客户端，前端使用 React、TypeScript、Tailwind CSS 和 Vite。
当前版本采用白色双栏工作区，顶部导航为「文译 / 同传 / 字幕 / 工具」。使用 #E8E8E8 细边框、12–14px 圆角、Inter 字体和黑色强调。输入区只保留面板外框，textarea 与其容器没有内层边框。
界面不包含品牌区、Apple 三色按钮、清空全文、术语库、场景选择、语音、导入文档或翻译操作按钮。

## 当前功能

- 悬停悬浮球时，工具栏像抽屉一样从悬浮球内向左拉出，跟随外观设置（浅色为白底、深色为深底），右上角显示关闭叉号；截图图标为 18px，悬停时变为樱桃色且不添加背景；抽屉高度为 40px、左侧圆角为 10px，并与球体贴合。抽屉使用遮罩和 GSAP 水平位移拉出、沿原路收回，与退出叉号共用时间轴，快速移入移出时从当前进度反向播放，退场完成后再收回原生点击区域；鼠标移到按钮区域时保持展开，离开后收起。剪刀入口当前仅提供界面。悬浮球与主窗口的叉号、Alt+F4 会询问关闭或最小化，支持取消和“以后不再提示”；勾选后记住本次选择，重启后仍生效，可在用户菜单“关闭时”改回每次询问。最小化保留状态并在后台运行，可从任务栏或再次打开 exe 恢复。
- 桌面端只允许运行一个实例，重复打开 exe 会唤起现有窗口，避免出现多个悬浮球。
- 首次启动时，悬浮球的球心位于屏幕宽度 75%、高度 25% 的位置。之后启动恢复上次退出时的悬浮球位置；打开工作区前也会保存球的位置，工作区移动不会覆盖它。位置保存在用户配置目录的 `lychee/floating-position.json`。
- 桌面端启动时先显示 56 × 56 逻辑像素的置顶悬浮球，可按住拖动；鼠标移入时圆底亮为白色、整体不透明度为 100%，荔枝图案使用明亮桃粉和嫩绿，移开 3 秒后变为参考图的深灰色并降低到 45% 不透明度，让屏幕内容透出，取消外圈阴影及暗态描边。颜色和透明度以 500 毫秒动画平滑过渡，重新移入会取消变暗倒计时。悬停全程使用固定的 118 × 66 透明画布，只切换工具区裁剪和元素动画，不移动或缩放 WebView；收起后预留工具区域不拦截桌面点击。过期的悬停回调不能覆盖新状态。双击后在当前显示器居中打开原有的 860 × 600 工作区并取消置顶，也支持聚焦悬浮球后按 Enter 或空格打开。关闭和最小化仍使用工作区原有按钮。
- 文译 / 同传 / 字幕 / 工具导航；文译提供双栏编辑区与原文、译文语言选择。
- 右上角用户菜单复刻参考图中的头像、额度、充值额度、积分明细、操作手册、设置和外观选项。浅色 / 跟随系统 / 深色会实际切换全局主题，选择保存在本机 `lychee.appearance.v1`；默认跟随系统。头像 Y 和 14,320 积分为截图展示数据，四个业务入口暂作占位，点击提示尚未开放，待接入账户服务。
- 文译两栏始终等高，默认占内容区一半高度；随原文增长同步增高，最多占内容区 90%，超出后在框内滚动，删短文字后自动缩回。
- 5,000 字输入限制、动态字数、复制与译文喜欢 / 取消喜欢；两栏间的细线双向箭头可交换原文与译文语言。
- 字幕页提供 Silero VAD 本地语音检测、WebSocket 实时转写与翻译、双语桌面字幕和实时预览，以及语言、监听来源、五种样式、字号和位置设置；自定义位置可拖动或用方向键微调。服务器缺少原文时只显示译文。工具页目前显示内容占位；截图翻译、图片翻译等功能归入工具页。
- 原文停止输入 600 毫秒后自动翻译；中文输入法组词期间不请求。切换标签会保留已发出的请求和译文，返回文译页不会重复请求；尚未发出的请求在返回后重新计时。修改内容、切换语言或点击重试会取消旧请求，旧译文不会覆盖新内容。
- 状态在本机持久化，字体随应用打包；窗口固定为 860 × 600 逻辑像素，禁用调整大小与最大化。Windows / Linux 在右上角显示最小化、关闭线条按钮，macOS 在左上角显示关闭、最小化圆点按钮；点击调用 Wails 窗口接口，标题栏空白区域可拖动窗口。
- 翻译页调用真实服务，支持等待、加载、错误和重试，复制按钮复制当前译文。原语言支持自动识别；同传尚未接入语音服务。

## 字幕音频监听

Windows 64 位桌面端在「字幕 → 监听来源」默认选择「全局监听」，并列出已打开桌面窗口的应用；不提供麦克风监听。旧版设置首次迁移时将来源设为全局监听，保留语言与字幕外观设置，后续记住用户选择。不要求先播放声音或建立音频会话。同时补充正在播放声音的后台应用，不使用已停止播放的残留音频会话判定应用是否打开。界面可见时每 2 秒刷新列表，恢复窗口或打开下拉框时也会刷新。选择应用后打开桌面字幕开关，界面会显示监听来源、等待 / 检测到声音和实时音量条。会议结束后应用主窗口仍打开时继续列出该应用，静音时显示等待声音。监听中切换来源会先停止旧采集；切换工作区标签保持监听，关闭开关或退出客户端释放资源。目标进程退出后停止监听并提示，应用重启后需要重新选择。

应用选项与选中项动态显示该进程 EXE 中的图标，无需预置图标素材。图标转换为透明的 32×32 PNG，通过本机 Wails 桥接传到界面；内存缓存最多 128 个可执行文件，文件修改后重新提取，读取或加载失败时显示通用应用图标。

同一应用进程及子进程只列出一项。多个独立的同名应用保留各自的监听目标，以可见窗口标题区分；标题相同或没有标题时使用实例编号。界面不显示 PID；长标题省略显示，悬停可查看完整名称。浏览器按进程树监听，当前不提供标签页选择。

应用列表过滤 Windows 窗口宿主、文字输入、桌面外壳、搜索等系统辅助进程，以及 Realtek 音频控制台；Windows `SystemApps` 与系统设置目录中的组件也不列出。窗口枚举跳过 DWM 隐藏的窗口。过滤同样应用于后台音频会话，避免系统组件播放声音后重新出现；保留普通桌面软件和商店安装的用户应用。

单独应用使用 Windows [进程回环采集](https://learn.microsoft.com/en-us/samples/microsoft/windows-classic-samples/applicationloopbackaudio-sample/)，包含目标进程及子进程，不采集其他应用的声音；需要 Windows 11 或 Windows build 20348 及以上。按相同可执行文件的父子进程合并浏览器等多进程应用，并用进程创建时间防止重启后的 PID 复用。全局监听使用默认输出设备的 [WASAPI loopback](https://learn.microsoft.com/en-us/windows/win32/coreaudio/loopback-recording) 采集混合声音，设备断开会显示错误。

开启字幕并成功连接音频来源后，桌面会显示独立的置顶字幕框。主窗口最小化或切换工作区时仍然显示；字幕框不抢焦点、穿透鼠标操作，不出现在任务栏。语言、字号、五种样式和顶部／底部／自定义位置与设置页同步，自定义位置通过预览拖动或方向键调整。字幕显示在开启时主窗口所在显示器的工作区内，适配 DPI 缩放并避开任务栏；关闭监听、切换来源或采集结束时清空字幕并收起窗口；退出应用释放资源。

桌面字幕显示服务器返回的真实内容，未监听时设置页仍提供双语示例。字幕画布固定为 1000 × 240 逻辑像素，按显示器 DPI 缩放，窄屏只按可用宽度收窄；文字增多、字号改变或缺失原文均不会撑大窗口。原文与译文各最多一行，长文本按实际字体宽度从头分页，只有译文时居中显示；服务器文本中的换行转为空格。

接收与显示使用独立缓冲：转写和临时译文先合并缓存，已定稿的原文／译文片段成对进入阅读队列。从头顺序呈现，默认约中文每秒 12 字、英文每秒 30 个字母。同一行能放下的后续片段立即追加，不设置段间固定停顿；超过任一语言的单行宽度时，当前行完整显示后停留约 0.6 秒，再整行清屏显示下一页。没有后续内容时，末段保留 2 秒；新内容到达后，能放在同一行就立即接续，否则按换页规则处理。持续播放期间，大量结果只增加待阅读内容，不覆盖未读文字、不自动加速；预览显示预计待阅读时间。可恢复的 ASR 修订错误会将收到但未定稿的预览标记为未完成，不能视为已结算结果。

连续约 2.5 秒没有声音时，视为当前播放暂时结束：丢弃当前显示片段之后的积压原译和临时预览，取消旧语音处理，保留监听开关；正在显示的片段仍完整呈现并停留 2 秒后清空。声音恢复后自动清空旧显示并使用新的字幕接收批次；旧会话迟到的结果不会再次显示，也不会混入新视频。检测使用低阈值的实际音频能量而非“有没有人说话”，短暂停顿和低音量播放不会立即清空。仅凭音频不能精确判断网页是否关闭；全局监听时其他应用的声音也会保持活跃，持续无声画面同样可能触发清理。手动关闭、切换来源和应用退出不等待静音宽限，直接清空。

连续定稿字幕可以跨语音会话在同一行接续，只呈现新增部分，不重播或重新淡入旧文字。单行宽度不足时整行换页，不截掉前文来容纳后续内容。原文和译文同步换页，字体、字号、DPI 和窄屏可用宽度共同决定容量。缺失原文状态变化或未完成片段单独显示，避免错接双语。

默认「深底白字」按参考图使用约 70% 不透明度的黑色底（约 30% 透出画面），保证白字可读性。其他预设保留轻量选择：暖黄描边为完全无底，轻雾白字、浅底深字、暗夜轻底分别使用约 10%、15%、20% 不透明度的背景。文字保持实色，并绘制细描边与轻阴影；取消背景模糊、边框和圆角。预览与桌面字幕的西文统一使用内置 Roboto Medium（500 字重）；桌面端仅在应用进程内加载，不修改系统字体。中文 Microsoft YaHei UI、日文 Yu Gothic UI、韩文 Malgun Gothic 使用系统字体，缺失时由系统回退。原生文字以两倍分辨率绘制后缩小，改善透明窗口中的抗锯齿；默认深底款使用更轻的描边。

采集输出为 16 kHz、单声道、16 位小端 PCM，每约 200ms 送入 Go 内的 Silero VAD，再以 32ms 帧按实时速度上传语音段；保留 320ms 前置音频与 1.28s 真实尾部静音，句中停顿不剪接。长时间无语音时不上传。原始 PCM 不经过前端、不写入文件；Wails `audio:level` 仅传递音量和本次监听编号，`audio:status` 提供生命周期，`subtitle:frame` 同步桌面和预览中的字幕。

语音端点默认由文本翻译地址派生为 `ws://127.0.0.1:8000/v1/translate/speech`。开启字幕前需在本机配置有效的 `LYCHEE_ACCESS_TOKEN`，也可通过 `LYCHEE_SPEECH_URL` 单独指定完整 WS(S) 地址。令牌仅由 Go 通过 Authorization 请求头发送，不写入 URL、前端存储或日志。客户端账户入口仍为占位，暂未提供 Steam 登录界面。自然语音段结束时发送 `finish` 并等待 `completed`；手动停止或持续静音则取消旧处理，不继续播放其积压结果。语言或来源变化会先结束旧会话。详细协议、Docker 启动、模型来源、边界与验证方法见 [字幕实现文档](docs/subtitles.md)。

本地开发库测试可运行 `powershell -ExecutionPolicy Bypass -File .\scripts\Start-SubtitleTest.ps1`：创建专用测试账号和 24 小时凭证、设置本机环境变量并启动 exe；后续运行复用有效凭证，`-RefreshToken` 可重新签发。仅用于明确允许写入测试数据的本机开发库，已有余额不会重置，服务器代码不变。

普通测试运行 `go test ./...`。原生隔离测试会短暂播放两种低音量测试音，并验证只收到目标应用的测试音、应用退出后停止采集：

```powershell
$env:LYCHEE_AUDIO_HARDWARE_TEST = '1'
go test -v -run TestApplicationAudioHardware -count=1 ./internal/app
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
wails build -ldflags '-X github.com/yanghui1-arch/lychee/internal/app.translationEndpoint=https://api.example.com/v1/translate'
```

`LYCHEE_TRANSLATE_URL` 的运行时配置优先于打包默认值。地址或路由变更只需修改该配置。桌面端由 Wails 的 Go HTTP 代理发送请求，浏览器开发预览由 Vite 代理发送请求，两者均通过同源 `/api/translate` 访问，避免服务端 CORS 配置差异。请求超时为 30 秒。

## 开发

需要 Go 1.25+、Bun、Wails CLI 2.15.0、Windows WebView2 Runtime，以及支持 CGO 的 GCC（例如 MinGW-w64）。Windows x64 的 Silero 模型与 ONNX Runtime 已放在 `models/`，随 exe 嵌入，无需另装 Python 或推理服务。

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
main.go                         Wails 启动入口与前端构建资源嵌入
internal/app/                   Go 后端实现与对应测试
  app.go                        Wails 窗口配置、服务组装与生命周期
  translation.go                可配置的翻译服务 HTTP 代理
  audio*.go                     Windows 回环采集、应用枚举与监听生命周期
  speech*.go                    语音流水线、协议、排序、恢复与字幕合并
  vad*.go                       Silero ONNX 本地推理与平台适配
  subtitle*.go                  字幕配置、阅读队列、分页与原生置顶窗口
models/                         模型与推理 DLL 嵌入、固定版本来源与授权
docs/subtitles.md               字幕部署、协议、资源边界与验证
build/                          平台资源和打包配置
frontend/
  assets_windows.go             桌面字幕共用字体与授权嵌入
  src/
    app/                        应用入口与全局行为
    components/ui/              上游 UI 组件与适配
    features/translator/        工作区、输入检测、翻译请求与状态
    features/subtitle/          字幕设置、音量、实时预览与桌面桥接
    lib/                        桌面桥接与通用工具
    styles/                     设计变量、布局和动效
  wailsjs/go/app/                Wails 生成的后端服务桥接
  wailsjs/runtime/               Wails 生成的运行时桥接
  licenses/                     上游组件与字体授权文本
```

Go 后端集中在 `internal/app/`，仅供当前应用使用；对应测试与实现放在同一目录。
开发和构建命令仍在仓库根目录运行。字体和模型由资源所在目录的 Go 包嵌入，仍随 exe 打包。

工作区状态集中在 `useTranslator.ts`，存储于本机 `lychee.workspace.v1`。
历史记录最多 20 条，收藏最多 50 条；浏览器预览和桌面运行使用各自的本机存储。
页面控件直接组合 shadcn 组件，功能目录只承担布局和业务状态。
翻译接口集中在 `translation-api.ts`，输入防抖、输入法与请求取消集中在 `useTranslation.ts`；工作区组件只组合控件与显示状态。

## 组件来源

- [GSAP](https://gsap.com/)：悬浮工具框和退出叉号使用可反向播放的时间轴；通过 gsap.context 在 React 卸载和 StrictMode 重建时清理动画。

- [Transitions.dev](https://transitions.dev/library.html)：顶部导航复用 Tabs sliding，译文操作复用 Like button 的心形填充、弹跳与八颗粒子动效，保留其默认粉红色选中效果。复制与保存提示使用 Toast open / close 的上浮、淡入、模糊和缩放。状态和键盘操作使用 shadcn/Base UI；菜单使用其动效参数。
- [Rare UI](https://www.rareui.com/components)：Animated Counter，保留原始实现和署名。
- [shadcn/ui](https://ui.shadcn.com)：Base Nova Tabs、Table、Input、Button、Toggle、Select、Dropdown Menu 与 Textarea，底层使用 Base UI。Tabs 的方向样式适配当前 Base UI 的 `data-orientation` 属性。
- [Base UI Toast](https://base-ui.com/react/components/toast)：管理提示的生命周期、自动关闭、重复消息与无障碍播报。
- [Base UI Avatar](https://base-ui.com/react/components/avatar)：用户头像回退字母；用户菜单沿用 shadcn Dropdown Menu，外观选择使用其 Radio Group / Radio Item，弹出箭头由 Base UI Menu Arrow 定位。
- [Fontsource](https://fontsource.org/fonts/inter)：统一使用自托管 Inter 可变字体，OFL 授权随源码保留；Inter 未覆盖的中文字形由系统无衬线字体补齐。
- [Roboto](https://github.com/googlefonts/roboto-3-classic/releases/tag/v3.015)：字幕使用自带的 Roboto Medium 3.015，预览与 Windows 原生窗口共用同一份 TTF。OFL 授权与下载来源记录保存在 `frontend/licenses/roboto-OFL.txt` 和 `roboto-SOURCE.txt`。

授权原文位于 `frontend/licenses/`，本 README 保留组件来源链接与署名。
Rare UI 使用 MIT + Commons Clause + Attribution 条款，具体授权范围以该目录中的原文为准。
