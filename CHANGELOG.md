# MaaEnd Client 更新日志

## v0.5.1 (2026-04-17)

### 维护

- 精简仓库测试文件，移除临时新增的回归用例以便快速发版。业务逻辑与 v0.5.0 完全一致。
- 版本号单一真相来源 `config.CurrentVersion` 升级至 `0.5.1`；`config.yaml`、README 示例同步刷新。

---

## v0.5.0 (2026-04-17)

### 修复

- **彻底修复 Agent 子进程无法承载 CustomAction / CustomRecognition 的问题**。此前 MEC 仅 `exec.Command` 启动 agent 子进程，未传入 AgentClient 分配的 identifier，导致子进程在 `os.Args[1]` 校验处立即退出（日志 `Usage: go-service <identifier>`），Pipeline 中所有 `custom_action` 在 MaaFW 侧全部命中 `Action is null`。
- **版本号由程序管理**：新增 `config.CurrentVersion` 常量作为单一真相来源，程序启动时若本地 `config.yaml` 的 `version` 字段与可执行文件内嵌版本不一致，会自动覆写并在日志中提示。避免升级二进制后本地旧配置文件造成"倒退"的版本上报。

### 新功能（PI 协议对齐）

- 按 MaaFW PI V2 标准集成 **AgentClient**：
  - 每个 agent 对应一个 `AgentClient` 实例，identifier 由 interface.json 显式指定或由 MaaFW 自动生成（参见 `core.AgentConfig.Identifier`，PI v2.1.0 可选字段）；
  - 启动顺序：`NewAgentClient → BindResource → 启动子进程（identifier 作为最后一位参数） → Connect(带 30s 超时) → 执行任务 → Disconnect → Destroy`；
  - 任一环节失败视为致命错误直接返回，不再静默继续执行。
- 按 PI v2.5.0 「Agent 子进程环境变量」约定注入：
  - `PI_INTERFACE_VERSION`、`PI_CLIENT_NAME`、`PI_CLIENT_VERSION`、`PI_CLIENT_LANGUAGE`、`PI_CLIENT_MAAFW_VERSION`、`PI_VERSION`；
  - `PI_CONTROLLER` / `PI_RESOURCE` 为紧凑 JSON 字符串，所有 `$` 前缀 i18n 字段已解析为展示文本（`core.ResolveControllerForEnv` / `ResolveResourceForEnv`）。
- 按 PI V2 协议实现 `controller.display_*` 字段支持：
  - 新增 `display_raw`、`display_long_side`、`display_short_side`；
  - 优先级 `display_raw > display_long_side > display_short_side`，三者都未声明时按协议默认短边 720；
  - 移除原硬编码 `SetScreenshotTargetLongSide(1280)`，改由协议字段驱动。

### 技术改进

- 资源清理顺序显式化：AgentClient → AgentServer 子进程 → Tasker → Resource → Controller，且 startAgents 失败时立即回滚已启动的 client 防止泄漏。
- AgentServer 子进程在 Stop 时等待 `cmd.Wait()` 返回，避免僵尸进程。
- 新增单元测试覆盖：i18n 字段解析、PI 环境变量构造、未知控制器/资源错误、空字段豁免。

---

## v0.4.0 (2026-03-06)

### 新功能

- 完成 P4 协议对齐：capabilities 增加 option 适用性字段（`option.controller` / `option.resource`）
- 前端任务与选项渲染改为严格按协议上下文过滤（controller/resource 双维度）
- 预设任务支持 `preset.task.enabled` 语义：前后端一致跳过禁用项

### 兼容性改进

- `default_case` 统一采用 `string[]` 语义（`select/switch` 取首项，`checkbox` 使用整组）

---

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
