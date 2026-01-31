# MaaEnd Client - 远程控制客户端

MaaEnd Client 是 MaaEnd 终末地小助手的远程控制客户端，允许用户通过 Web 界面远程控制本地运行的 MaaEnd 执行自动化任务。

## 功能特性

- **远程控制**：通过 Web 界面远程下发任务、停止任务
- **实时状态**：实时上报任务执行进度和日志
- **实时截图**：支持远程获取游戏画面截图
- **自动重连**：断线自动重连，指数退避策略
- **多控制器**：支持 Win32、ADB 等多种控制方式
- **多资源**：支持官服、B 服等多种游戏资源

## 系统要求

- **操作系统**：Windows 10+ / Linux / macOS
- **Go 版本**：1.21+（开发编译）
- **MaaEnd**：需要安装 MaaEnd v1.6.0+
- **网络**：需要能够连接到云端服务器

## 快速开始

### 1. 下载

从 [Releases](https://github.com/your-repo/releases) 下载对应平台的可执行文件。

### 2. 放置

将 `maaend-client` 可执行文件放置到 MaaEnd 安装目录，或任意位置。

### 3. 运行

```bash
# 自动检测 MaaEnd 路径
./maaend-client

# 指定 MaaEnd 路径
./maaend-client -maaend /path/to/MaaEnd

# 指定服务器地址
./maaend-client -server ws://your-server.com/ws/maaend
```

### 4. 绑定设备

首次运行时，需要在 Web 端获取绑定码，然后在客户端输入绑定码完成绑定。

```
========================================
     MaaEnd Client - 远程控制客户端
========================================

设备未绑定，请按以下步骤操作：
1. 在 Web 端获取绑定码
2. 输入绑定码后按回车

请输入绑定码: 123456
```

绑定成功后，设备令牌会自动保存，下次启动自动认证。

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-c` | 配置文件路径 | `./config.yaml` |
| `-maaend` | MaaEnd 安装路径 | 自动检测 |
| `-server` | 服务器 WebSocket 地址 | `ws://localhost:15618/ws/maaend` |
| `-bind` | 绑定码（首次绑定时使用） | - |
| `-debug` | 调试模式 | `false` |

### 使用示例

```bash
# 使用自定义配置文件
./maaend-client -c /path/to/config.yaml

# 指定 MaaEnd 路径和服务器
./maaend-client -maaend D:/MaaEnd -server wss://api.example.com/ws/maaend

# 使用命令行绑定码（无需交互输入）
./maaend-client -bind 123456

# 调试模式
./maaend-client -debug
```

## 配置文件

配置文件 `config.yaml` 位于程序运行目录，首次运行会自动生成。

```yaml
# MaaEnd Client 配置文件

# 客户端版本号
version: "0.2.1"

server:
  # 云端 WebSocket 地址
  ws_url: "ws://localhost:15618/ws/maaend"
  # 连接超时
  connect_timeout: 10s
  # 心跳间隔
  heartbeat_interval: 30s
  # 重连最大延迟
  reconnect_max_delay: 30s

maaend:
  # MaaEnd 安装目录（为空则自动检测）
  path: ""
  # 覆盖 Win32 窗口类名（正则，留空不覆盖）
  win32_class_regex: ""
  # 覆盖 Win32 窗口标题（正则，留空不覆盖）
  win32_window_regex: ""

device:
  # 设备名称（为空则使用主机名）
  name: ""
  # 设备令牌（首次绑定后自动保存）
  token: ""

logging:
  # 日志级别: debug, info, warn, error
  level: "info"
  # 日志文件（为空则输出到控制台）
  file: ""
```

### 配置说明

| 配置项 | 说明 |
|--------|------|
| `version` | 客户端版本号，用于版本追踪 |
| `server.ws_url` | 云端服务器 WebSocket 地址 |
| `server.connect_timeout` | WebSocket 连接超时时间 |
| `server.heartbeat_interval` | 心跳发送间隔 |
| `server.reconnect_max_delay` | 断线重连最大等待时间 |
| `maaend.path` | MaaEnd 安装目录，为空时自动检测 |
| `maaend.win32_class_regex` | 覆盖窗口类名匹配规则（正则表达式） |
| `maaend.win32_window_regex` | 覆盖窗口标题匹配规则（正则表达式） |
| `device.name` | 设备显示名称，默认使用主机名 |
| `device.token` | 设备认证令牌，绑定后自动保存 |
| `logging.level` | 日志级别 |
| `logging.file` | 日志输出文件，为空输出到控制台 |

### Win32 窗口匹配

如果 MaaEnd 默认的窗口匹配规则无法找到游戏窗口，可以通过配置覆盖：

```yaml
maaend:
  # 匹配 UnityWndClass 类的窗口
  win32_class_regex: "UnityWndClass"
  # 匹配包含 "终末地" 的窗口标题
  win32_window_regex: ".*终末地.*"
```

## MaaEnd 路径自动检测

客户端会按以下顺序检测 MaaEnd 安装路径：

1. 当前目录（检查是否存在 `interface.json` 和 `maafw` 目录）
2. 当前目录的父目录
3. `%APPDATA%/MaaEnd`（Windows）
4. 常见安装位置：`C:/MaaEnd`, `D:/MaaEnd`, `E:/MaaEnd`

如果自动检测失败，请使用 `-maaend` 参数手动指定。

## 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│                        Web 前端                              │
│  1. 生成绑定码                                               │
│  2. 下发任务指令                                             │
│  3. 实时显示状态/日志/截图                                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      云端服务器                              │
│  - 管理设备绑定关系                                          │
│  - 转发任务指令                                              │
│  - 推送状态更新                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    MaaEnd Client                             │
│  1. WebSocket 连接云端                                       │
│  2. 接收任务指令                                             │
│  3. 调用 MaaFramework 执行任务                               │
│  4. 上报状态/日志/截图                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     MaaFramework                             │
│  - 控制器：Win32 / ADB                                       │
│  - 资源：官服 / B 服                                         │
│  - 任务：日常、签到、自动化等                                 │
└─────────────────────────────────────────────────────────────┘
```

## 版本信息

客户端会上报两个版本号：

| 版本 | 来源 | 说明 |
|------|------|------|
| `maaend_version` | MaaEnd `interface.json` | MaaEnd 核心版本 |
| `client_version` | `config.yaml` | 客户端版本 |

## 常见问题

### Q: 提示 "未找到 MaaEnd 安装目录"

A: 请确保 MaaEnd 安装目录包含 `interface.json` 文件和 `maafw` 目录，或使用 `-maaend` 参数指定正确路径。

### Q: 提示 "未找到匹配的窗口"

A: 请确保游戏已启动，并检查 MaaEnd 的窗口匹配配置。可以在 `config.yaml` 中配置 `win32_class_regex` 和 `win32_window_regex` 来覆盖默认规则。

### Q: 设备显示离线

A: 检查网络连接，确保能够访问云端服务器。客户端会自动尝试重连。

### Q: 绑定码无效

A: 绑定码有效期为 5 分钟，请在 Web 端重新生成绑定码。

### Q: 任务执行失败

A: 查看控制台日志了解详细错误信息。常见原因：
- 游戏窗口未找到
- 资源加载失败
- 任务配置不正确

## 开发指南

详见 [DEVELOPMENT.md](./DEVELOPMENT.md)

## 许可证

MIT License
