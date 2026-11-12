# 本地语音检测资源

资源已下载并随 Windows x64 exe 嵌入。用户无需联网下载模型，也无需在 exe 旁复制此目录。

| 文件 | 固定来源 | SHA-256 |
| --- | --- | --- |
| `silero_vad.onnx` | [Silero VAD v6.2](https://github.com/snakers4/silero-vad/blob/v6.2/src/silero_vad/data/silero_vad.onnx) | `1a153a22f4509e292a94e67d6f9b85e8deb25b4988682b7e174c65279d8788e3` |
| `onnxruntime.dll` | [Microsoft ONNX Runtime 1.22.0 / Windows x64 CPU](https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-win-x64-1.22.0.zip)，压缩包内 `lib/onnxruntime.dll` | `579b636403983254346a5c1d80bd28f1519cd1e284cd204f8d4ff41f8d711559` |

Silero 使用 MIT 授权，原文见 `silero-LICENSE`。ONNX Runtime 使用 MIT 授权，原文与第三方声明分别见 `onnxruntime-LICENSE`、`onnxruntime-NOTICES.txt`。

Go 绑定固定为 `github.com/yalue/onnxruntime_go v1.22.0`，与 DLL 的 C API 版本一致。模型以 16 kHz、512 个采样一帧推理，输入为 `input [1,576]`（64 个前文采样 + 512 个新采样）、`state [2,1,128]`、`sr [1]`；输出为 `output [1,1]` 和 `stateN [2,1,128]`。每次监听创建独立状态和张量，CPU 推理限制为一个线程。

运行时将嵌入的 DLL 和上述授权声明解包到本机用户缓存 `Lychee/onnxruntime-1.22-<hash>/`，校验 DLL 的 SHA-256 后通过绝对路径加载；模型直接从内存加载。缓存只含推理库与授权，不保存音频、令牌或转写。更新模型或 DLL 时应同时更新来源、哈希与对应推理测试。

## 截图 OCR 模型

`ocr/` 内为预先下载的 PP-OCRv6 Small 检测和识别模型，随 exe 嵌入，无需用户首次使用下载。来源：
- https://huggingface.co/PaddlePaddle/PP-OCRv6_small_det
- https://huggingface.co/PaddlePaddle/PP-OCRv6_small_rec

Apache 2.0 授权见 `ocr/LICENSE`，模型说明保留在各模型目录的 README.md。模型文件哈希见 `ocr/SHA256SUMS`。更新模型时需同时更新哈希及真实推理检查。

运行时将模型解包到用户缓存 `Lychee/pp-ocrv6-small-<hash>/ocr/`，逐文件比较 SHA-256，文件缺失或损坏时从 exe 恢复。推理显式指定本地模型目录，并禁用模型源连通性检查和 Hugging Face 在线访问。OCR Python 运行环境仍由 `scripts/setup-ocr.ps1` 预先安装到 exe 旁的 `ocr-runtime`。

### 版面检测试验

额外内置 `ocr/PP-DocLayout-S`（来源 https://huggingface.co/PaddlePaddle/PP-DocLayout-S ，Apache 2.0，保留上游 README），与 OCR 在同一 Python 进程异步加载；不启用完整 PP-StructureV3。模型和配置均预下载，SHA-256 记录在上述清单。离线加载显式传入模型目录。
