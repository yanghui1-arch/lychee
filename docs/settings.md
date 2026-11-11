# 外观与本地设置

外观设置控制工作区和悬浮工具栏的浅色、深色及系统主题。各功能的语言、输入和样式分别保存在本机，不依赖账户同步。

## 主题切换

在工作区右上角用户菜单选择浅色、跟随系统或深色，默认跟随系统。`useAppearance` 读取偏好，结合 `prefers-color-scheme` 切换根元素的 `dark` 类；跟随系统时系统主题变化即时生效。样式通过全局主题变量和 Tailwind 状态表达。存储不可用时仍允许当前页面切换。

实现：[useAppearance.ts](../frontend/src/app/useAppearance.ts)、[UserMenu.tsx](../frontend/src/features/translator/UserMenu.tsx)、[globals.css](../frontend/src/styles/globals.css)。

## 用户菜单与业务占位

头像字母 Y、14,320 积分来自静态展示数据，不表示当前登录身份或真实余额。充值额度、积分明细、操作手册和设置入口仅显示“功能暂未开放”提示；菜单中的外观切换已实现。当前没有账户登录界面，字幕所用令牌由 Go 从环境读取，与菜单展示数据无关。

菜单由 shadcn Dropdown Menu / Base UI Menu 组成，头像使用 Base UI Avatar 回退文字，外观使用单选组，提示使用 Base UI Toast。

实现：[UserMenu.tsx](../frontend/src/features/translator/UserMenu.tsx)、[dropdown-menu.tsx](../frontend/src/components/ui/dropdown-menu.tsx)、[toast.tsx](../frontend/src/components/ui/toast.tsx)。

## 存储与迁移

| 位置 / 键 | 保存内容 | 校验与边界 |
| --- | --- | --- |
| localStorage `lychee.appearance.v1` | `light`、`system`、`dark` | 非法值回退到跟随系统 |
| localStorage `lychee.workspace.v1` | 原文、两端语言、预留历史收藏、字号和历史偏好 | 原文最多 5000 单位；历史 20、收藏 50；字号仅 14、16、18 |
| localStorage `lychee.subtitle.v2` | 字幕语言、监听来源、字号、样式、位置 | 字号整数 18–40，坐标百分比 0–100；来源为 system 或应用标识 |
| localStorage `lychee.interpret.v1` | 听说语言、设备、触发方式与快捷键 | 语言和触发方式校验，非法值回退默认值 |
| 用户配置目录 `lychee/floating-position.json` | 悬浮球坐标 | 两个整数；越界时恢复初始位置 |

字幕读取不到 v2 时尝试 v1：保留有效语言和外观，监听来源重置为全局，随后保存 v2。字幕默认英文转简体中文、全局监听、30 号字、深底白字、底部位置，自定义坐标默认 50% / 82%。

各读取器对损坏 JSON 或无效字段回退到默认设置；localStorage 写入失败时仍保持当前会话可操作。浏览器预览与桌面使用各自存储环境。清除对应键会在下次读取时恢复默认值；这不涉及服务端账户或积分。音频与字幕队列仅保留在内存，不保存为录音；原文及预留记录属于本地保存内容。

实现：[useTranslator.ts](../frontend/src/features/translator/useTranslator.ts)、[字幕 model.ts](../frontend/src/features/subtitle/model.ts)、[InterpretWorkspace.tsx](../frontend/src/features/interpret/InterpretWorkspace.tsx)、[window_position.go](../internal/window_position.go)。

## 验证

检查三种外观、系统主题变化、刷新和重启恢复、损坏存储回退及字幕 v1 迁移。喜欢按钮是临时状态，其边界见[文本翻译](translation.md)。
