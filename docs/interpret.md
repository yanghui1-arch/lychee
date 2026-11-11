# 同传设置

同传页面为“听别人说”和“我说话”的翻译方向、设备及触发方式提供配置界面。目前仅实现设置、设备枚举和快捷键录入，尚未接入采集、ASR、翻译或语音合成；开始操作提示服务尚未接入。需要真实声音翻译时使用[实时桌面字幕](subtitles.md)。

## 语言与设备

“听别人说”默认自动识别原语言、译为简体中文，输出使用默认扬声器；“我说话”默认简体中文译为 English，输入使用麦克风、输出使用默认扬声器。

设备通过 `navigator.mediaDevices.enumerateDevices()` 枚举，监听 `devicechange` 更新。列表保留默认设备选项，仅加入有标识和可见名称的实际设备；浏览器未暴露设备信息或枚举失败时保留默认项，不会因为列出设备而启动录音。已保存但暂时不存在的设备显示“所选设备当前不可用”，避免静默替换。

## 触发方式与快捷键

提供“语音活动”和“按键触发”两种设置，默认语音活动。快捷键录入期间捕获页面键盘事件：

- 忽略重复按键和单独的修饰键。
- 接受带 Ctrl / Alt / Meta 的组合，或 F1–F12；Shift 可作为组合中的附加修饰键。
- 录入后保存规范化显示字符串，例如 `Ctrl + Shift + A`，并结束录入。
- Backspace / Delete 清空，Esc 取消，Tab 结束录入并保留正常焦点切换。

这里保存的是页面设置字符串，未注册系统全局同传快捷键；截图 F1 属于独立的原生快捷键功能。

## 持久化与代码位置

设置自动写入 `lychee.interpret.v1`，读取时验证语言、设备字符串和触发方式。录入中的临时状态不持久化。设备查询在页面挂载时运行，卸载时移除事件监听。

| 代码 | 职责 |
| --- | --- |
| [InterpretWorkspace.tsx](../frontend/src/features/interpret/InterpretWorkspace.tsx) | 设置模型、默认值、设备枚举、快捷键录入、分组界面和服务占位提示 |
| [settings.tsx](../frontend/src/components/ui/settings.tsx) | 设置组、行、下拉和分段选择控件 |
| [model.ts](../frontend/src/features/translator/model.ts) | 共用语言列表及语言类型 |
| [TranslatorWorkspace.tsx](../frontend/src/features/translator/TranslatorWorkspace.tsx) | 同传标签入口与页面挂载 |

## 验证

检查语言切换、设备变化、设备不可用回显、快捷键接受与拒绝规则、取消与清除、重启后的设置恢复。开始操作仍应明确提示尚未接入服务。
