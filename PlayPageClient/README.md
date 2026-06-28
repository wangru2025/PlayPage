# PlayPageClient

PlayPage 原生客户端工程骨架。

## 架构

- `PlayPage.Core`：共享业务层，负责 API、认证、项目、上传、急救站、AI 圆桌等逻辑。不能引用任何 UI 框架。
- `PlayPage.Windows`：WinForms UI，优先使用 Windows 标准控件，方便 NVDA/Narrator 读取。
- `PlayPage.Android`：.NET for Android UI，使用传统 Android View，不使用 Compose/MAUI 自绘 UI。

## 本地开发

Windows 端：

```powershell
dotnet build .\src\PlayPage.Windows\PlayPage.Windows.csproj
```

Android 端建议交给 GitHub Actions 构建，本机不需要安装 Android SDK/Java。

## 无障碍约定

- Windows UI 优先使用 WinForms 标准控件，并设置 `AccessibleName`、`AccessibleDescription`、合理的 `TabIndex`。
- Android UI 优先使用传统 View，并设置 `ContentDescription`，状态变化时发送 accessibility announcement。
- 不用自绘按钮、自绘列表、皮肤控件作为核心交互。
- Core 层不弹窗、不播报、不依赖 UI，只返回结构化结果或抛出中文异常。
