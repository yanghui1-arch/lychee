# 桌面截图

截图功能将桌面区域捕获为图片，并提供标注、复制、保存、置顶及逐屏滚动拼接。截图翻译按钮目前仅占位。

## 入口与窗口

Windows 桌面版可点击悬浮工具栏的剪刀按钮，也可在其他应用中按 **F1** 开始截图。文译、同传、字幕页面不再提供截图按钮。

双击悬浮球会打开独立工作区，悬浮球继续显示；再次双击只显示并激活同一个工作区。工作区右上角 `×` 隐藏该窗口，再次双击悬浮球可以继续使用。F1 截图只临时隐藏悬浮球，保留工作区供截图；退出截图后恢复悬浮球。从托盘退出荔枝会同时结束工作区。

## 选区与调整

进入截图时默认预览任务栏之外的整个工作区，不自动保存或置顶。单击可直接使用完整工作区选区，按下时保持原预览，不会出现临时的 0×0 或闪烁。拖动选择矩形区域，松手即显示标注工具栏；不需要二次确认。蓝色边框向内绘制并加粗，贴近屏幕两侧也能看到；绘制时与刚画完时不显示控制点，单击选区后才显示八个控制点供调整。工具栏跟随选区，并自动避让屏幕边缘。
## 图形与文字标注

矩形、圆形支持调整大小及七种颜色，绘制过程中和刚画完时不显示控制点，单击已有图形就可以移动或显示控制点调整大小，无需取消当前工具；按住 Alt 可在已有图形内继续绘制新标注。箭头从鼠标按下的位置指向松手的位置，不显示缩放控制点；支持颜色及小、中、大三种粗细，默认使用小号箭头，使用实心圆点直接选择。选择文字工具后单击选区即可输入，输入时显示虚线矩形，完成后虚线消失；支持颜色及小/中/大字号，三档按钮直接选择；Enter 换行，单击框外完成输入。输入框随内容自动增高，不显示滚动条。
## 工具状态与编辑历史

再次点击当前工具图标可取消该工具；无论当前使用哪个工具，都可直接选择已有标注并移动。颜色、字号、粗细选项属于各自工具，仅在使用对应工具时出现在主工具栏下方，不再显示全局配置栏。退出按钮与其它图标始终在同一主工具行。Delete 删除选中的标注。Ctrl+Z 撤销、Ctrl+Y 或 Ctrl+Shift+Z 恢复选区和标注操作，包括添加、移动、缩放、删除及样式修改；新操作会清空恢复分支。滚动拼接切换到新的长图后重新记录编辑历史。
## 保存、复制与桌面贴图

保存按钮导出包含标注的 PNG，并保持与其它工具按钮一致的中性样式。复制按钮或编辑器中的 Enter 会把选区与标注作为真正的图片写入 Windows 剪贴板，成功后恢复原窗口，可在聊天框使用 Ctrl+V 粘贴；复制失败保留编辑器供重试。文字输入框中的 Enter 仍用于换行。置顶按钮将截图作为可拖动的桌面贴图显示；右上角叉号或 Esc 关闭贴图并恢复原来的窗口。
- 翻译按钮仅占位，尚未实现翻译。
## 滚动拼接与限制

滚动截图每点击一次向选区下方的应用滚动并追加一屏，自动去除重叠部分；到达底部或无法匹配时给出提示。请框选实际可滚动的内容区域，避开浏览器地址栏、侧栏、固定导航及动态视频。完成滚动后再标注更方便。长图最大高度 30000 px。
- 当前截取主屏幕，一次编辑或置顶一张截图。F1 被其他应用占用时会在应用日志中记录注册失败，工具栏入口仍可用。

## 功能验证

先在 `frontend/` 目录运行 `bun run dev -- --host 127.0.0.1`，打开 `/tests/screenshot-harness.html`，在浏览器开发者工具中执行：

```js
await (await import('/tests/screenshot-functional.js')).runScreenshotFunctionalTests()
```

该场景使用真实浏览器的 SVG、Canvas、PointerEvent 和 React 交互，模拟桌面桥接以覆盖默认工作区预览、无确认框选、调整尺寸、全部标注工具、虚线文字输入、撤销/恢复、颜色/字号/粗细、PNG、滚动入口、置顶及取消。原生截图与 F1 必须用 `wails build` 后的桌面程序验证。滚动图像去重与接缝另有 `TestScrollingScreenshotStitchesOnlyNewContent` 场景测试。

复制图片的格式检查：`go test ./internal/screenshot -run TestScreenshotClipboardImage -count=1`。此检查验证 Windows DIB 的尺寸、颜色和方向，不会改写系统剪贴板。浏览器场景模拟桌面桥接以验证复制按钮、Enter 和失败重试交互。

## 实现与代码位置

前端使用冻结的屏幕图像作为编辑底图，选区和标注作为编辑状态；导出时将选区及标注合成为图片。Wails 桥接负责原生捕获、文件对话框、剪贴板与桌面窗口，音频、翻译工作区不参与图像编辑。

| 代码 | 职责 |
| --- | --- |
| [ScreenshotEditor.tsx](../frontend/src/features/screenshot/ScreenshotEditor.tsx) | 选区和标注交互、控制点、文字编辑、样式、撤销恢复、导出与工具栏 |
| [model.ts](../frontend/src/features/screenshot/model.ts) | 图像、几何和标注类型与计算 |
| [screenshot.css](../frontend/src/features/screenshot/screenshot.css) | 编辑器专用样式 |
| [App.tsx](../frontend/src/app/App.tsx) | screenshot:start / end / error 事件及编辑器切换 |
| [window.go](../internal/window.go) | 向前端暴露截图服务方法 |
| [service.go](../internal/screenshot/service.go)、[screenshot.go](../internal/screenshot/screenshot.go) | 截图生命周期、前端帧、保存复制及滚动入口 |
| [screenshot_windows.go](../internal/screenshot/screenshot_windows.go) | 主屏捕获、原生窗口、F1、滚动输入与覆盖区域 |
| [screenshot_context.go](../internal/screenshot/screenshot_context.go) | 截图运行上下文 |
| [screenshot_clipboard_windows.go](../internal/screenshot/screenshot_clipboard_windows.go) | PNG 解码、DIB 转换及原生图片剪贴板 |
| [screenshot_stitch.go](../internal/screenshot/screenshot_stitch.go) | 相邻画面重叠检测、接缝与长图合成 |

复制失败保留编辑器，避免丢失标注；保存取消或失败不应视为已输出图片。滚动模式将输入交给下方应用，再捕获新一屏并匹配重叠区域，匹配失败时提示，不能保证对固定导航或动态画面正确拼接。滚动切换到长图后重新建立编辑历史，原先操作不能跨底图撤销。

工具栏使用 Lucide 图标、实底和细边框，按工具展开相关选项；仅保留短颜色反馈并遵从减少动态效果。视觉整理参考 [unslop](https://heydamir.design/unslop)，作者 Damir Sokolovsky（[@heydamir](https://x.com/heydamir)），CC BY 4.0。
