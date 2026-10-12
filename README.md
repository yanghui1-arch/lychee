# 荔枝 · Lychee

基于 Wails v2 的桌面翻译客户端，前端使用 React、TypeScript、Tailwind CSS 和 Vite。
当前版本采用白色双栏工作区，顶部导航为「文译 / 图译 / 同传 / 字幕」。使用 #E8E8E8 细边框、12–14px 圆角、Inter 字体和黑色强调。输入区只保留面板外框，textarea 与其容器没有内层边框。
界面不包含品牌区、Apple 三色按钮、清空全文、术语库、场景选择、语音、导入文档或翻译操作按钮。

## 当前功能

- 文译 / 图译 / 同传 / 字幕导航；文译提供双栏编辑区与原文、译文语言选择。
- 文译两栏始终等高，默认占内容区一半高度；随原文增长同步增高，最多占内容区 90%，超出后在框内滚动，删短文字后自动缩回。
- 5,000 字输入限制、动态字数、复制与译文喜欢 / 取消喜欢；两栏间的细线双向箭头可交换原文与译文语言。
- 图译、字幕页目前显示内容占位。
- 原文停止输入 600 毫秒后自动翻译；中文输入法组词期间不请求。切换标签会保留已发出的请求和译文，返回文译页不会重复请求；尚未发出的请求在返回后重新计时。修改内容、切换语言或点击重试会取消旧请求，旧译文不会覆盖新内容。
- 状态在本机持久化，字体随应用打包；窗口固定为 860 × 600 逻辑像素，禁用调整大小与最大化。Windows / Linux 在右上角显示最小化、关闭线条按钮，macOS 在左上角显示关闭、最小化圆点按钮；点击调用 Wails 窗口接口，标题栏空白区域可拖动窗口。
- 翻译页调用真实服务，支持等待、加载、错误和重试，复制按钮复制当前译文。原语言支持自动识别；同传尚未接入语音服务。

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
- [Fontsource](https://fontsource.org/fonts/inter)：统一使用自托管 Inter 可变字体，OFL 授权随源码保留；Inter 未覆盖的中文字形由系统无衬线字体补齐。

授权原文位于 `frontend/licenses/`，本 README 保留组件来源链接与署名。
Rare UI 使用 MIT + Commons Clause + Attribution 条款，具体授权范围以该目录中的原文为准。
