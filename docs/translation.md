# 文本翻译

文译将输入文本提交给翻译服务，在双栏工作区显示原文和译文。翻译由服务端完成，客户端负责输入、语言参数、请求状态和结果操作。

## 输入与语言

- 原文最多 5,000 个 JavaScript UTF-16 代码单元；输入框 `maxLength` 和动态计数均以字符串 `length` 为准，emoji 等字符可能占多个单位。
- 原语言默认自动识别，目标语言默认 English。提供简体中文、繁体中文、美国英语、英国英语、日语、韩语、法语、德语，实际发送 `zh-CN`、`zh-TW`、`en-US`、`en-GB`、`ja-JP`、`ko-KR`、`fr-FR`、`de-DE`。
- 中间箭头只交换两端语言，不交换文本。原语言为自动识别时禁用交换。
- 两栏保持等高，初始约占内容区 50%，根据输入高度增长至约 90%，随后内部滚动；删短文字后缩回。窄视口采用上下布局。

实现：[TranslatorWorkspace.tsx](../frontend/src/features/translator/TranslatorWorkspace.tsx)、[LanguageSelect.tsx](../frontend/src/features/translator/LanguageSelect.tsx)、[model.ts](../frontend/src/features/translator/model.ts)、[useTranslationPanelHeight.ts](../frontend/src/features/translator/useTranslationPanelHeight.ts)。

## 自动翻译与状态

输入非空文本并停止修改 600ms 后发送请求，无需点击翻译按钮。中文输入法 composition 期间保持等待，组词结束后重新计时。纯空白文本回到空闲状态。

状态依次为 `idle`、`waiting`、`loading`、`success` 或 `error`。等待和加载时显示提示，错误时显示原因及重试按钮；重试针对当前文本和语言重新请求。

修改文本、修改语言、输入法状态变化或重试会通过 AbortController 取消旧请求，已取消请求的结果不再写回。切换到其他标签时，已发出的请求继续完成并保留结果；返回文译不重复请求。尚未发出的防抖请求会被清除，返回后重新等待 600ms。组件卸载时取消在途请求。

实现：[useTranslation.ts](../frontend/src/features/translator/useTranslation.ts)。

## 请求与错误

前端对同源 `/api/translate` 发起 POST：

```json
{"text":"你好","source_language":"auto","target_language":"en-US"}
```

成功响应读取字符串字段 `translated_text`。服务错误、格式错误及网络失败由请求适配层转换为界面消息。桌面版由 Go 反向代理请求上游，保留配置地址的路由和查询参数；只接受 `/api/translate` 的 POST，其他路径返回 404，其他方法返回 405。代理超时 30 秒，上游不可达返回 502，超时返回 504。开发服务使用 Vite 代理同一路径。

地址通过 `LYCHEE_TRANSLATE_URL` 或打包默认值配置，具体命令见[README 配置](../README.md#配置)。语音令牌的注入位于语音链路，文本代理未实现同样的主动 Bearer 注入。

实现：[translation-api.ts](../frontend/src/features/translator/translation-api.ts)、[translation.go](../internal/translation/translation.go)、[vite.config.ts](../frontend/vite.config.ts)。

## 复制与喜欢

只有翻译成功且译文非空时启用复制和喜欢。复制使用当前真实译文，通过桌面剪贴板或浏览器适配写入，成功显示“已复制”，失败显示重试提示。喜欢仅是当前工作区的 React 状态，输入或语言修改后清除，不持久化、不写入收藏列表、不调用服务端。

实现：[TranslatorWorkspace.tsx](../frontend/src/features/translator/TranslatorWorkspace.tsx)、[desktop.ts](../frontend/src/lib/desktop.ts)、[like-button.tsx](../frontend/src/components/ui/like-button.tsx)、[toast.tsx](../frontend/src/components/ui/toast.tsx)。

## 本地状态与预留记录

输入、语言和偏好由 `useTranslator` 写入 `lychee.workspace.v1`。读取时验证语言、记录格式和字号，文本截断至上限；读写失败仍可在当前会话使用。

代码还保留历史、收藏及 RecordsPanel：历史上限 20 条、收藏上限 50 条，以文本和两端语言判重；初始历史为静态演示数据。当前工作区没有历史或收藏入口，也没有在成功翻译后自动写历史。预留 `saveHistory` 写入的结果仍为占位文本。因此这些代码不代表已开放的真实翻译记录功能。

实现：[useTranslator.ts](../frontend/src/features/translator/useTranslator.ts)、[RecordsPanel.tsx](../frontend/src/features/translator/RecordsPanel.tsx)、[model.ts](../frontend/src/features/translator/model.ts)。

## 验证

前端测试位于 [frontend/tests/](../frontend/tests/)，代理测试位于 [translation_test.go](../internal/translation/translation_test.go)。修改后重点检查防抖、输入法、切换标签、取消旧请求、重试和真实译文复制。
