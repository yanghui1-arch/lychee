# 桌面截图

截图功能将桌面区域捕获为图片，并提供标注、复制、保存、置顶及逐屏滚动拼接。截图翻译使用本地 PP-OCRv6 Small 检测和识别，再调用现有文本翻译服务。

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
- 翻译：点击翻译图标，默认翻译成简体中文。最多并发 4 个文字区域，先完成的结果先覆盖，失败区域保留原文；再次点击翻译恢复原图。复制、保存和置顶均包含译文。背景使用区域边缘颜色填充，译文自动换行缩小；复杂背景与倾斜文字暂不提供图像修复。先完成滚动截图，再在新长图中翻译。
- Windows 首次准备：安装 Python 3.10–3.12，执行 `powershell -ExecutionPolicy Bypass -File scripts/setup-ocr.ps1`，在 exe 旁生成 `ocr-runtime`。检测和识别模型已预先下载并嵌入 exe，运行时从 exe 解包到带内容哈希的用户缓存目录并校验完整性；不依赖 Paddle 全局模型缓存，不在首次使用时联网检查或下载模型；只将文字发送到 `LYCHEE_TRANSLATE_URL`。自定义运行环境可设置 `LYCHEE_OCR_PYTHON` 为 python.exe 绝对路径。OCR 脚本嵌入 exe，目标机器需安装 Python 并运行准备脚本生成 `ocr-runtime` 虚拟环境。模型固定 `PP-OCRv6_small_det` 和 `PP-OCRv6_small_rec`，依赖固定 paddleocr 3.7.0 / paddlepaddle 3.3.0。
- Esc 取消截图与当前 OCR/翻译。单次 OCR 超时 3 分钟，文字翻译沿用服务器代理的 30 秒超时。
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

截图翻译检查：`bun test tests/screenshot-translation.test.ts`（在 frontend 目录）验证并发上限、乱序完成和单块失败隔离。`go test ./internal/ocr` 验证无效输入；设置 `LYCHEE_OCR_PYTHON` 和 `LYCHEE_OCR_TEST_IMAGE` 后，`go test ./internal/ocr -run TestSmallModelIntegration -v -count=1` 使用真实模型，测试图片需包含 Lychee。Windows CPU 禁用 oneDNN，以规避 Paddle 3.3 的 PIR 属性兼容问题。模型接口参考 [PaddleOCR 官方文档](https://www.paddleocr.ai/main/en/version3.x/pipeline_usage/OCR.html)。

## OCR 预加载与诊断日志

悬浮球主进程启动时，在后台启动常驻 Python OCR 进程，加载本地 PP-OCRv6 Small。工作区子进程不重复加载。预加载不阻塞窗口显示；首次截图如尚未就绪则等待，后续截图复用同一个模型实例。OCR 按序推理，识别后的翻译仍最多并发 4 块。取消正在执行的 OCR 会结束该工作进程以避免串帧，下一次请求重新加载；取消等待不影响后台预加载。退出应用会结束 OCR 工作进程。

日志目录：`%LOCALAPPDATA%/Lychee/logs/`，文件为 `screenshot-日期-进程ID.jsonl`。单文件超过 20 MiB 时保留一份 `.1` 备份；旧日期、旧进程日志可手工清理。

- `ocr_preload`：总加载耗时，以及模型文件准备、Python 依赖导入、模型初始化耗时。包含进程启动与通信的整体时间，不把它称为纯推理时间。
- `ocr`：使用同一 `session` 串联一次截图任务，记录等待就绪/排队、OCR 请求耗时、纯推理耗时、识别区域数量及成功/失败/取消状态。
- `translation_region`：区域下标（从 0 开始）、翻译请求耗时与结果状态。此耗时包含客户端到服务端的往返和 JSON 解析，不代表服务端纯模型推理速度。
- `first_region_painted`、`screenshot_translation_complete`：首次译文准备覆盖与任务整体耗时、完成数、失败数。

默认只记录计时和状态，不记录截图、原文、译文或令牌。需要检查识别/翻译内容时，启动前设置 `LYCHEE_SCREENSHOT_DEBUG=1`，或使用：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/Start-SubtitleTest.ps1 -ScreenshotDebug
```

详细模式在 `ocr` 的 `details` 中记录区域文字、坐标、置信度，在 `translation_region` 中记录原文和译文；令牌和图片仍不写入日志。已有客户端需先退出，使新进程继承调试开关。

## 版面分析试验

增加预下载并内置的 PP-DocLayout-S，和 PP-OCRv6 Small 共用常驻 Python。识别后将文字框按覆盖率匹配到最小的文本版面块；只在同一版面块内，根据行距、对齐、字号及时间戳/用户名边界组合成翻译单元。无法匹配版面块的文字仍独立翻译。版面检测失败时回退到原始文字框，不阻断 OCR。它主要面向文档，聊天界面分块效果仍需实际截图检验。

组合后的文字一次翻译，返回译文后按照原始文字框排版覆盖，保留框之间的空白，不用整块矩形覆盖其它内容。每组仍可并发翻译。

新增日志：
- `layout_preload`：版面模型加载耗时、加载前后进程 RSS 和差值；`ocr_preload` 同时列出 `layout_load_ms`。
- `layout_analysis`：版面预测总耗时（包含前后处理）、每块的类别/坐标/置信度、原始文字框数量、分组数量、`groups` 的 `block` / `members` 映射（下标从 0 开始，未匹配块为 null）。详细模式还记录 `raw_regions` 与 `grouped_regions` 文字及原始框坐标，便于检查误合并/漏合并。
- `metrics`：版面推理前后 RSS、OCR 后 RSS、进程历史 RSS 峰值。`*_mb` 字段使用 MiB（1024² 字节），是同进程测量值，峰值并非版面模型独占内存。
- `outcome=fallback`：版面分析不可用或失败，继续逐框翻译。成功 OCR 现在正确记录为 success，不再被正常清理时的 context cancel 误标记。

分组场景测试：`python -m unittest discover -s internal/ocr -p test_grouping.py`；前端测试验证合并译文只填充原始框且保留完整文字。

## 译文字号与溢出

译文根据同组原始文字框的中位行高及 Canvas 字体字面高度估算字号，组内统一，不再为容纳长译文而逐块缩小。保留标题与正文的大小差异。仅凭截图估算，不能精确恢复源字体字号。

优先按原宽度换行，向下扩展至下一处横向相交的 OCR 文字框或截图选区边界；不覆盖相邻文字。仍放不下的组保留原图文字，在工具栏下方的展开区域显示完整译文；保存、复制和置顶时该组仍是原图文字。`translation_render` 日志记录成功/overflow，以及估算字号 `font_size_px` 与排版行数。翻译按合并后的文字组请求一次，未合并的框仍各自请求。

## 文字像素保护修复

版面块作为合并参考；模型漏检续行时，允许依据原始行左边界、行距与消息边界合并。同一条消息的用户名保持原图，正文合并后翻译。

启用 OCR `return_word_box`，按词级对齐提取正文位置；无法定位词的区域保留原图。擦除仅限这些位置中保守提取的文字像素，不再填充矩形背景。彩色前景保留，低置信度单字符误识别跳过。译文在独立 Canvas 上排版，若新字形触碰未确认的前景像素，则整组保留原图并提供展开译文。未知复杂背景同样优先回退；词级定位和像素分类仍有识别误差，不能当作通用语义分割。

`translation_render.outcome` 区分 success、insufficient_space、protected_content、missing_text_mask。OCR metrics 记录 recognized_text_pixels 与 word_alignment_enabled；详细日志里的 erase 是对应文字像素的行程掩膜。未识别内容不会再因为位于行级检测框内就被整块清除。

## 框内重绘修复

文字能按原字号排入可用框就绘制，不再以单个像素色差或缺少擦除掩膜否决整组。掩膜只决定哪些旧文字像素被清理；译文字形单独合成，并裁剪在批准的文字排版槽内。没有掩膜也允许绘制，原内容中无法安全清理的部分可能残留，不再误报空间不足。小数坐标在栅格化时取整，避免掩膜索引错位。排版允许使用右侧可用空间；只有真正未排完字符才返回 insufficient_space。界面提示对应真实的空间不足。

靠右短标签（如 Ad）允许按译文所需宽度向左借用空白，最多两个字号宽度，遇到已有前景或文字框即停止，不再只向右寻找空间。渲染日志增加 available_width_px、text_characters、placed_characters。

## 重影修复

正文清理包含彩色 ClearType 抗锯齿边缘，按局部实际背景色恢复，不再只擦除灰度文字像素。词级坐标有估计误差，清理支持范围带少量边缘余量并覆盖相邻词缝，限制在已对齐的正文附近，用户名与外围内容不参与清理。合成读取范围包含完整擦除掩膜，避免边缘被裁掉；OCR 裁图与叠加位置统一整数像素。无法识别的零散原文仍可能保留，避免把未知内容直接当作正文抹掉。

## 零散残留修复

已识别的纯标点 token（引号、星号等）纳入正文清理，不再因为不含字母数字就跳过。单独识别为小行的闭合引号/星号，在几何关系符合时附加到前一条消息的清理掩膜，不单独请求翻译，也不占用译文排版槽。文字边缘清理余量适当扩大，背景差阈值从 8 降为 4，以消除细小 ClearType 残留。范围仍受词级位置与用户名边界限制，未识别的其它前景不做全图去噪。

空间计算现在跳过同组正文清理掩膜中的旧字像素，避免把即将擦掉的词边缘当作右侧/左侧障碍；段落每一行使用自己对应的背景色，不再统一使用首行颜色，以支持悬停底色变化。LOL→哈哈场景回归覆盖“20px原框+旧词边缘→应扩展后一行显示”。

排版按目标字体实际字形高度（actualBoundingBoxAscent/Descent）计算行槽，不再用字号×1.25作为硬性容纳高度。实际绘制使用 alphabetic 基线对齐字形顶部。只略超出单行的短译文最多横向收紧15%，保持字号高度；Canvas maxWidth 限制字形在框内。渲染日志增加 ink_height_px、line_height_px。

译文按实际字形高度在原文字行区域内垂直居中，多行作为一个文字块居中；空间允许时上下各留至少1px抗锯齿余量。保留 alphabetic 基线、原字号高度及水平对齐，避免字形顶边贴住裁剪边界。

## 文章标题与界面元素分组修复

不同字号（行高比例低于 0.8）、日期、标签不再跨行合并；非聊天内容同行合并距离收紧。CamelCase 账号标识及日期保留原图。评论气泡误识别的 Q/O 仅在“数字 + comments/replies”模式中作为装饰剔除，不作为文字请求翻译。日志 metrics 增加 preserved_metadata_count 和 grouping_policy_version=2。

正文清理依据前景连通组件恢复完整笔画，避免近似词框边界把同一个字裁成残笔；明显大于文字高度的图形不参与恢复。文章样图回归验证标题独立保持31px行高，顶部图片、作者头像/名称/日期、橙色图标逐像素不变。

分组策略 v3：已确认用户名/时间戳开头的聊天消息，允许续行 OCR 框高度比例低至0.6；普通文章仍保持0.8阈值。避免正文续行因检测框高度波动被拆成两个翻译/排版单元，标题与正文仍保持隔离。

## 带冒号标题与多词作者名

策略 v4：聊天前缀仅接受紧凑用户名/时间戳形式，多词标题中的冒号不再作为“用户名：正文”分隔符。落在标题版面块中的短标签冒号也不当作用户名。日期邻接的短作者名保留原图；日期 OCR 漏检时，使用相邻头像与标题上方位置作为附加证据，不再把 Earl Grey 这类多词作者名作为正文翻译后换行。

样图验证完整的 A Small Laptop…Build Kit:…Chromebook, Claude, and GitHub 作为同组翻译内容；作者、头像和日期像素保持不变。聊天用户名保护和文章标题分组均有回归检查。

点击翻译后立即锁定本次截取框，OCR/翻译进行中、完成、失败以及恢复原图后，都不能移动、缩放、重新框选或改为滚动截图。撤销/恢复仅还原标注，不能改变锁定区域；需要新选区时退出截图后重新截取。

置顶截图在原选区的位置向右下偏移 12 像素，并保留原尺寸；坐标使用捕获像素并叠加显示器原点，避免 DPI 缩放时再次转换。超过屏幕的长图按比例缩小，越界时将窗口移回屏幕内。
