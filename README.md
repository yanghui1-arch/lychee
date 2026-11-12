# 荔枝

荔枝是一款桌面工具应用，旨在将日常工作中常用的工具集中到桌面，方便随时调用。目前提供文本翻译、实时双语字幕、截图标注与滚动拼接等功能，通过悬浮球快速打开工作区或使用工具。

# 支持功能

- [文本翻译](docs/translation.md)：停止输入后自动翻译，支持语言切换、复制、错误重试。
- [实时桌面字幕](docs/subtitles.md)：监听应用或全局声音，本地语音检测，实时转写、翻译及双语字幕。
- [桌面截图](docs/screenshot.md)：区域截图、标注、撤销恢复、PNG 保存、图片复制、置顶及滚动拼接。
- [悬浮球与窗口](docs/desktop.md)：独立工作区、托盘、位置恢复及后台运行。
- [外观与本地设置](docs/settings.md)：浅色、深色、跟随系统以及各功能设置持久化。

[同传](docs/interpret.md)目前提供设置界面，尚未接入语音服务；截图翻译、账户和积分业务入口也尚未开放。功能文档索引见 [docs/README.md](docs/README.md)。

# 配置

文本翻译和实时字幕需要连接相应服务，截图等本地工具可独立使用。使用翻译相关功能时，在运行前设置以下环境变量，修改后重新启动应用：

| 配置项 | 默认值 | 用途 |
| --- | --- | --- |
| `LYCHEE_TRANSLATE_URL` | `http://127.0.0.1:8000/v1/translate` | 文本翻译的完整 HTTP(S) 地址，包含路由 |
| `LYCHEE_SPEECH_URL` | 从文本地址派生，默认 `ws://127.0.0.1:8000/v1/translate/speech` | 字幕语音服务的完整 WS(S) 地址 |
| `LYCHEE_ACCESS_TOKEN` | 无 | 文本、截图翻译及字幕连接所需的有效登录令牌 |

```powershell
$env:LYCHEE_TRANSLATE_URL = 'http://127.0.0.1:8000/v1/translate'
$env:LYCHEE_ACCESS_TOKEN = '<有效 access_token>'
# 语音服务独立部署时设置：
$env:LYCHEE_SPEECH_URL = 'ws://127.0.0.1:8000/v1/translate/speech'
```

文本地址的运行时环境变量优先于打包默认值。发布时可设置默认服务地址：

```sh
wails build -ldflags '-X github.com/yanghui1-arch/lychee/internal/translation.translationEndpoint=https://api.example.com/v1/translate'
```

接口格式见[文本翻译](docs/translation.md)，字幕服务、凭证及开发测试脚本见[实时桌面字幕](docs/subtitles.md)。界面设置自动保存在本机，存储位置见[外观与本地设置](docs/settings.md)。

# 开发

需要 Go 1.25+、Bun、Wails CLI 2.15.0、WebView2 Runtime，以及支持 CGO 的 GCC。模型与推理运行库已嵌入应用，来源和授权见 [models/README.md](models/README.md)。

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
bun install --cwd frontend --frozen-lockfile
go mod download
wails dev
```

检查与构建：

```sh
bun run --cwd frontend typecheck
bun run --cwd frontend format:check
bun test --cwd frontend tests
go test ./...
go vet ./...
wails build
```

构建产物为 `build/bin/lychee.exe`。`wails.json` 配置前端安装、开发和构建命令；`frontend/dist/`、`frontend/node_modules/`、`build/bin/` 为构建输出。样式约定见 [frontend/STYLING.md](frontend/STYLING.md)。

## 技术栈

| 技术 / 框架 / 组件 | 使用位置与作用 |
| --- | --- |
| Go、Wails v2 | `internal/app.go` 组装服务、资源代理与生命周期；Wails 桥接前端和原生窗口、音频、截图服务 |
| React 19、TypeScript、Vite | `frontend/src/` 的工作区、设置和截图编辑器；Vite 提供开发服务与前端打包 |
| Tailwind CSS 4、CVA、clsx、tailwind-merge | 页面样式、组件变体与 `frontend/src/lib/utils.ts` 的类名合并；主题变量在 `styles/globals.css` |
| shadcn/ui（Base Nova）、Base UI | `components/ui/` 的 Tabs、Button、Textarea、Select、Dropdown Menu、Toggle、Table、Input；用于导航、文译及设置。Base UI Avatar 用于用户菜单，Toast 用于操作提示 |
| GSAP | `app/useFloatingControlsAnimation.ts`：悬浮工具栏与关闭图标的可反向时间轴 |
| Motion、Transitions.dev 动效参考 | `components/ui/like-button.tsx` 的喜欢动效，`tabs.tsx` 的滑动导航与 `toast.tsx` 的提示动效；`settings.tsx` 的分段选择与 `animated-counter.tsx` 的数字动画也使用 Motion；`app/App.tsx` 配置减少动态效果 |
| Rare UI Animated Counter | `components/ui/animated-counter.tsx`：文译输入字数动画 |
| Lucide React | 导航、窗口操作、翻译操作、截图工具及设置图标 |
| Fontsource Inter、Roboto Medium | Inter 用于应用界面；打包 Roboto TTF 由字幕预览和原生字幕共用，中日韩字形使用系统字体回退 |
| WASAPI、go-wca、go-ole、Win32 | `internal/audio/` 的声音采集、应用枚举及图标；`internal/window*` 和 `internal/subtitle/` 的原生窗口 |
| Silero VAD、ONNX Runtime Go | `internal/vad/` 进行本地语音检测；模型与 DLL 由 `models/` 嵌入 |
| Gorilla WebSocket | `internal/speech/` 的鉴权、语音上传与转写翻译事件接收 |
| uniseg | `internal/subtitle/` 的字素分割，分页时保留 emoji 和组合字符 |
| kbinani/screenshot、原生剪贴板接口 | `internal/screenshot/` 捕获屏幕、导出图片、写入 DIB 剪贴板及拼接滚动截图 |
| energye/systray | `internal/tray_windows.go` 的托盘显示与退出菜单 |

组件和字体授权保留在 [frontend/licenses/](frontend/licenses/)，模型授权保留在 [models/](models/)。

## 项目结构

```text
main.go                         启动入口与前端资源嵌入
internal/
  app.go                        服务组装、Wails 配置及生命周期
  window*.go、workspace*.go      悬浮球、独立工作区、窗口区域与位置
  tray_windows.go               托盘与退出
  translation/                  文本翻译 HTTP 代理
  audio/                        应用枚举、图标与声音采集
  vad/                          语音检测与前后缓冲
  speech/                       WebSocket、语音会话、排序与恢复
  subtitle/                     字幕合并、阅读分页与原生渲染
  screenshot/                   截图、剪贴板、快捷键与滚动拼接
frontend/
  src/app/                      应用入口、悬浮球、外观及动画
  src/features/translator/      文译、语言选择及请求状态
  src/features/interpret/       同传设置界面
  src/features/subtitle/        字幕设置、预览与音频桥接
  src/features/screenshot/      截图选区、标注及编辑历史
  src/components/ui/            通用 UI 组件
  src/lib/                      桌面适配及工具函数
  src/styles/                   全局主题和字体入口
  wailsjs/                      Wails 生成的服务与运行时绑定
  tests/                        前端测试与浏览器交互场景
models/                         模型、推理库、资源嵌入与授权
scripts/                        本机开发测试辅助脚本
docs/                           按功能组织的说明与代码索引
build/                          应用资源及打包配置
wails.json                      Wails 项目配置
```
