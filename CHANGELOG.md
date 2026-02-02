# MaaEnd Client 更新日志

## v0.3.0 (2026-02-02)

### 新功能

- **支持 interface.json 的 import 机制**：现在可以正确解析 MaaEnd 的外部任务导入功能，支持从 `import` 字段指定的外部 JSON 文件中加载任务和选项配置
- **支持 switch 选项类型**：新增对 `switch` 类型选项的支持，功能与 `select` 类似，通常用于 Yes/No 二选一场景
- **支持 default_check 任务属性**：正确解析任务的 `default_check` 字段，用于标识默认选中的任务

### 修复

- **修复分辨率转换问题**：添加 `SetScreenshotTargetLongSide(1280)` 设置，确保截图正确缩放到 MaaEnd 资源设计的标准分辨率（1280x720 基准），解决了非 16:9 屏幕下 ROI 坐标越界的问题
- **修复选项默认值逻辑**：优化 `resolveSelectOption` 的默认值选择逻辑，支持 `Default` 字段作为备选默认值

### 技术改进

- Win32 和 ADB 控制器创建后自动设置截图目标长边为 1280，保证与 MaaEnd 桌面版行为一致
- 改进外部任务文件的解析和合并逻辑

---

## v0.2.2 (2025-12-15)

### 功能

- 初始版本发布
- WebSocket 客户端连接和心跳维护
- MaaFramework 封装和任务执行
- 设备绑定和认证机制
- 任务状态实时推送
- 截图功能支持
- Win32 和 ADB 控制器支持
- 本地凭证存储
