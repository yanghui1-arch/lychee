# 本地语音检测资源

资源已下载并随 Windows x64 exe 嵌入。用户无需联网下载模型，也无需在 exe 旁复制此目录。

| 文件 | 固定来源 | SHA-256 |
| --- | --- | --- |
| `silero_vad.onnx` | [Silero VAD v6.2](https://github.com/snakers4/silero-vad/blob/v6.2/src/silero_vad/data/silero_vad.onnx) | `1a153a22f4509e292a94e67d6f9b85e8deb25b4988682b7e174c65279d8788e3` |
| `onnxruntime.dll` | [Microsoft ONNX Runtime 1.22.0 / Windows x64 CPU](https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-win-x64-1.22.0.zip)，压缩包内 `lib/onnxruntime.dll` | `579b636403983254346a5c1d80bd28f1519cd1e284cd204f8d4ff41f8d711559` |

Silero 使用 MIT 授权，原文见 `silero-LICENSE`。ONNX Runtime 使用 MIT 授权，原文与第三方声明分别见 `onnxruntime-LICENSE`、`onnxruntime-NOTICES.txt`。

Go 绑定固定为 `github.com/yalue/onnxruntime_go v1.22.0`，与 DLL 的 C API 版本一致。模型以 16 kHz、512 个采样一帧推理，输入为 `input [1,576]`（64 个前文采样 + 512 个新采样）、`state [2,1,128]`、`sr [1]`；输出为 `output [1,1]` 和 `stateN [2,1,128]`。每次监听创建独立状态和张量，CPU 推理限制为一个线程。

运行时将嵌入的 DLL 和上述授权声明解包到本机用户缓存 `Lychee/onnxruntime-1.22-<hash>/`，校验 DLL 的 SHA-256 后通过绝对路径加载；模型直接从内存加载。缓存只含推理库与授权，不保存音频、令牌或转写。更新模型或 DLL 时应同时更新来源、哈希与对应推理测试。
