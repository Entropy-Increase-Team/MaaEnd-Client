# MEC 标准化改造 TODO（基于 MaaFW ProjectInterface V2）

> 目标：把 MEC 从“针对 MaaEnd 的项目适配器”升级为“遵循 MaaFW 标准协议的通用远程控制终端”。
>
> 核心原则：**以标准文档为源头实现（spec-first）**，而不是以某个项目当前行为为实现依据（project-first）。

---

## 一、方向纠偏（来自讨论结论）

当前被指出的关键问题不是“能不能跑”，而是“实现路径是否正确”。

- 错误路径：按 MaaEnd 当前实现去对齐，容易形成“项目耦合逻辑”，可维护性和通用性差。
- 正确路径：按 MaaFW `ProjectInterface` 规范实现协议引擎，MaaEnd 只是其中一个可运行项目。

一句话：

**先实现协议标准，再兼容项目差异，而不是反过来。**

参考标准：
- [MaaFramework - 3.3 Project Interface V2 协议](https://maafw.com/docs/3.3-ProjectInterfaceV2)

---

## 二、总体目标

### 2.1 功能目标

1. MEC 完整支持 PI v2.3.1 的核心语义（解析、激活、合并、执行）。
2. 同一套 MEC 内核可驱动多个遵循 PI 的项目（不仅 MaaEnd）。
3. 前后端能力上报与执行行为一致、可验证、可回归测试。

### 2.2 工程目标

1. 协议模型、执行编译器、运行器解耦。
2. 增加标准兼容测试集（正例 + 反例）。
3. 提供兼容性说明和迁移指引。

---

## 三、实施 TODO（按优先级）

## P0：基线与差异收敛（先判断，再动刀）

- [x] 建立 PI v2.3.1 基线清单（字段、类型、规则、版本增量）。
- [x] 输出 MEC 差异矩阵（支持/部分支持/未支持/行为偏差）。
- [x] 形成“必须修复项”与“可延期项”分级。

**验收标准**
- 可一眼看到每个字段和规则的实现状态。
- 评审可直接据此排期，不再凭感觉改。

---

## P1：解析层标准化（Parser）

- [x] `agent` 支持 `object | object[]` 两种形态。
- [x] `default_case` 支持 `string | string[]`（尤其 checkbox）。
- [x] 补齐 `global_option`。
- [x] 补齐 `resource.option`、`controller.option`。
- [x] 补齐 `option.resource`（当前已有 `option.controller`，需对齐资源维度）。
- [x] 统一 import 合并策略（task/option/preset）与冲突覆盖规则。

**验收标准**
- 文档示例能被稳定解析。
- 对同一配置，解析结果与文档语义一致。

---

## P2：规则引擎标准化（激活 + 合并）

- [x] 实现 option 激活判定：按 `controller/resource` 条件过滤。
- [x] 支持嵌套 `option.option` 的递归激活判定。
- [x] 实现标准合并顺序：
  - `global_option` < `resource.option` < `controller.option` < `task.option`
- [x] 保证后合并覆盖前合并（同名字段覆盖语义一致）。

**验收标准**
- 与文档给出的覆盖顺序与激活规则一致。
- 可通过单测复现复杂嵌套场景。

---

## P3：执行链路标准化（Compiler + Runner）

- [x] 引入“任务参数编译器”：
  - 输入：task + options + context(controller/resource)
  - 输出：最终 `pipeline_override`
- [x] 将“编译”与“执行（Tasker）”解耦，便于测试与替换。
- [x] 统一错误类型与可观测日志（解析错误、激活失败、冲突覆盖、执行失败）。

**验收标准**
- 不连 MaaFW 也可单测编译结果。
- 接入 MaaFW 后行为与编译输出一致。

---

## P4：能力上报与前后端一致性

- [x] 扩展 capabilities 上报，覆盖 option 适用性和默认值语义。
- [x] 前端严格按协议数据渲染，不写项目特判分支。
- [x] 历史/状态/任务类型展示统一走协议映射。

**验收标准**
- 前端展示与后端能力上报一致。
- 增加新 PI 字段时，不需要“硬编码项目名”改 UI。

---

## P5：测试体系与发布文档

- [ ] 建立“标准兼容测试集”（官方示例 + 自造反例）。
- [ ] 增加回归测试（每次改动自动校验关键语义）。
- [ ] 发布《MEC ProjectInterface 兼容性说明》。
- [ ] 发布迁移指南：从“项目适配逻辑”迁移到“协议驱动逻辑”。

**验收标准**
- 发布前跑完整兼容测试，无高优先级失败项。
- 文档可指导外部项目接入。

---

## 四、里程碑建议

- M1（1 周）：完成 P0 + P1（先把数据模型和解析打稳）。
- M2（1~2 周）：完成 P2 + P3（实现规则引擎和编译执行解耦）。
- M3（1 周）：完成 P4 + P5（联调、测试、文档发布）。

---

## 五、非目标（当前阶段不做）

- 不做“只对 MaaEnd 有用”的临时硬编码修补。
- 不做绕过协议语义的“能跑就行”捷径实现。
- 不将项目私有字段混入标准核心模型。

---

## 六、落地原则（开发守则）

1. **Spec-first**：先看标准条目，再写代码。
2. **测试先行**：先写语义测试，再实现。
3. **可解释性**：每个行为都能在文档中找到依据。
4. **可回归**：新增功能不破坏既有协议语义。

---

## 七、当前状态

- [x] 已形成标准化改造路线
- [x] 已完成 P0（基线清单 + 差异矩阵）
- [x] 已完成 P1（解析层标准化）
- [x] 已完成 P2（规则引擎激活与合并）
- [x] 已完成 P3（执行链路标准化）
- [x] 已完成 P4（能力上报与前后端一致性）
- [x] 已完成 P6.1 PI v2.5.0 Agent 子进程集成（AgentClient + PI_* 环境变量）
- [x] 已完成 P6.2 PI V2 `controller.display_*` 字段协议驱动
- [ ] 进行中：P5（测试体系与发布文档）
- [ ] 待办 P6.3：PI v2.4.0 顶层 `group[]` 声明 + `task.group` 字段（capability 上报与前端分组展示）
- [ ] 待办 P6.4：`focus` 回调机制（`display: log/toast/notification/dialog/modal` 渠道 + 占位符替换）
- [ ] 待办 P6.5：节点级事件（`Node.PipelineNode.*` / `Node.Recognition.*` / `Node.Action.*`）日志转发
- [ ] 待办 P6.6：Win32 新输入方式显式映射（`SendMessageWithWindowPos` / `PostMessageWithWindowPos`）+ `Background` 组合宏

## 八、v0.5.x 改动摘要（2026-04-17）

v0.5.1：精简仓库测试文件（不含业务逻辑改动）。

v0.5.0：

| 改动点 | 协议依据 | 文件 |
|---|---|---|
| `AgentConfig.Identifier` 字段 | PI V2 `agent.identifier`（可选，用于 socket 标识符） | `core/interface_parser.go` |
| `ControllerConfig.Display{Raw,LongSide,ShortSide}` | PI V2 `controller.display_*` 三者互斥 | `core/interface_parser.go` |
| `ResolveControllerForEnv` / `ResolveResourceForEnv` | PI v2.5.0 `PI_CONTROLLER` / `PI_RESOURCE` 要求 i18n 已解析的紧凑 JSON | `core/i18n_resolver.go` |
| `BuildAgentEnv` | PI v2.5.0「Agent 子进程环境变量」 | `maa/agent_env.go` |
| `AgentServer.Start(..., identifier, env)` | MaaFW agent-server 示例：`os.Args[1]` 为 identifier | `maa/agent.go` |
| `Wrapper.startAgents` 走 `AgentClient` 完整生命周期 | maa-framework-go `agent_client.go` 标准流程 | `maa/wrapper.go` |
| `applyScreenshotResolution` 替换硬编码 1280 | PI V2 `display_*` 协议字段 | `maa/wrapper.go` |
| `CurrentVersion` 单一真相来源 + 启动时覆写配置文件 | Client 版本号上报准确性 | `config/config.go` |

