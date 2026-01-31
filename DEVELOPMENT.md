# MaaEnd Client 开发文档

本文档面向开发者，介绍 MaaEnd Client 的架构设计、代码结构和开发指南。

## 目录结构

```
maaend-client/
├── main.go                 # 程序入口
├── config.yaml             # 配置文件（运行时生成）
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖锁定
├── README.md               # 用户文档
├── DEVELOPMENT.md          # 开发文档
│
├── client/                 # WebSocket 客户端
│   ├── client.go           # 客户端核心逻辑
│   ├── handler.go          # 消息处理器
│   └── protocol.go         # 消息协议定义
│
├── config/                 # 配置管理
│   └── config.go           # 配置加载/保存
│
├── core/                   # 核心解析器
│   ├── capabilities.go     # 设备能力构建
│   ├── interface_parser.go # interface.json 解析
│   └── option_resolver.go  # 任务选项解析
│
├── maa/                    # MaaFramework 封装
│   ├── wrapper.go          # 主封装类
│   ├── controller.go       # 控制器管理
│   ├── resource.go         # 资源管理
│   ├── task.go             # 任务执行
│   ├── callback.go         # 事件回调
│   └── agent.go            # Agent 服务
│
└── store/                  # 本地存储
    └── store.go            # 凭证存储
```

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.24+ |
| WebSocket | gorilla/websocket | 1.5.1 |
| 配置管理 | spf13/viper | 1.18.2 |
| MaaFramework | maa-framework-go | 3.5.1 |

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                           main.go                                │
│  - 解析命令行参数                                                 │
│  - 加载配置                                                      │
│  - 初始化各模块                                                   │
│  - 处理退出信号                                                   │
└─────────────────────────────────────────────────────────────────┘
                                │
            ┌───────────────────┼───────────────────┐
            ▼                   ▼                   ▼
    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
    │   config/    │    │   client/    │    │    maa/      │
    │  配置管理     │    │ WebSocket    │    │ MaaFramework │
    └──────────────┘    └──────────────┘    └──────────────┘
                                │                   │
                                ▼                   ▼
                        ┌──────────────┐    ┌──────────────┐
                        │    store/    │    │    core/     │
                        │  本地存储     │    │  解析器      │
                        └──────────────┘    └──────────────┘
```

### 模块职责

#### 1. client/ - WebSocket 客户端

负责与云端服务器的 WebSocket 通信。

**client.go** - 客户端核心
- 连接管理（连接、断开、重连）
- 消息发送队列
- 心跳维持
- 回调管理

**handler.go** - 消息处理
- 处理服务器下发的消息
- 分发到对应处理函数
- 调用 MaaWrapper 执行任务

**protocol.go** - 协议定义
- 消息类型常量
- 消息结构体定义
- 序列化/反序列化

#### 2. config/ - 配置管理

负责配置文件的加载、保存和自动检测。

**config.go**
- 使用 Viper 加载 YAML 配置
- 命令行参数覆盖
- MaaEnd 路径自动检测
- 配置文件格式维护

#### 3. maa/ - MaaFramework 封装

封装 MaaFramework Go SDK，提供简化的接口。

**wrapper.go** - 主封装类
- 初始化 MaaFramework
- 管理控制器、资源、Tasker
- 执行任务、截图

**controller.go** - 控制器
- Win32 控制器创建
- ADB 控制器创建
- 窗口匹配逻辑

**callback.go** - 事件回调
- 任务事件处理
- 状态上报
- 日志上报

**agent.go** - Agent 服务
- 启动 Agent 子进程
- 用于自定义识别器等高级功能

#### 4. core/ - 核心解析器

解析 MaaEnd 的 interface.json 配置。

**interface_parser.go**
- 解析 interface.json
- 获取任务、控制器、资源配置
- 路径处理

**capabilities.go**
- 构建设备能力信息
- 任务列表
- 选项信息

**option_resolver.go**
- 解析用户选择的任务选项
- 生成 pipeline_override

#### 5. store/ - 本地存储

管理设备凭证的本地存储。

**store.go**
- 保存/加载设备 Token
- JSON 文件存储

## 核心流程

### 1. 启动流程

```
main()
  │
  ├─► 解析命令行参数
  │
  ├─► 加载配置 (config.Load)
  │     ├─► 读取 config.yaml
  │     ├─► 设置默认值
  │     └─► 自动检测 MaaEnd 路径
  │
  ├─► 初始化本地存储 (store.NewStore)
  │     └─► 加载已保存的 Token
  │
  ├─► 初始化 MaaFramework (maa.NewWrapper)
  │     ├─► 加载 interface.json
  │     └─► 初始化 MaaFW SDK
  │
  ├─► 创建 WebSocket 客户端 (client.NewClient)
  │     └─► 设置回调函数
  │
  └─► 运行客户端 (client.Run)
        ├─► 连接服务器
        ├─► 认证/注册
        └─► 进入主循环
```

### 2. 认证流程

```
连接成功
  │
  ├─► 有 Token ─────► 发送 auth 消息
  │                       │
  │                       ├─► authenticated ─► 上报能力
  │                       │
  │                       └─► auth_failed ─► 清除 Token，等待绑定
  │
  └─► 无 Token ─────► 等待用户输入绑定码
                          │
                          └─► 发送 register 消息
                                  │
                                  └─► registered ─► 保存 Token，上报能力
```

### 3. 任务执行流程

```
收到 run_task 消息
  │
  ├─► 解析任务配置
  │
  ├─► 连接控制器 (ConnectController)
  │     ├─► 获取控制器配置
  │     ├─► 查找匹配窗口
  │     └─► 创建并连接控制器
  │
  ├─► 加载资源 (LoadResource)
  │     ├─► 获取资源路径
  │     └─► 加载资源包
  │
  ├─► 创建 Tasker
  │     ├─► 绑定控制器
  │     ├─► 绑定资源
  │     └─► 注册回调
  │
  └─► 执行任务 (RunTask)
        │
        ├─► 遍历任务列表
        │     ├─► 解析选项 (ResolveTaskOptions)
        │     ├─► 执行任务 (PostTask)
        │     └─► 上报状态 (task_status)
        │
        └─► 完成 ─► 上报完成 (task_completed)
```

## WebSocket 协议

### 消息格式

所有消息采用 JSON 格式：

```json
{
  "type": "消息类型",
  "payload": { ... },
  "timestamp": "2026-01-31T16:00:00+08:00"
}
```

### Client → Server 消息

| 类型 | 说明 | Payload |
|------|------|---------|
| `register` | 设备注册 | `RegisterPayload` |
| `auth` | 设备认证 | `AuthPayload` |
| `ping` | 心跳 | - |
| `capabilities` | 能力上报 | `CapabilitiesPayload` |
| `task_status` | 任务状态 | `TaskStatusPayload` |
| `task_log` | 任务日志 | `TaskLogPayload` |
| `task_completed` | 任务完成 | `TaskCompletedPayload` |
| `screenshot` | 截图上报 | `ScreenshotPayload` |

### Server → Client 消息

| 类型 | 说明 | Payload |
|------|------|---------|
| `registered` | 注册成功 | `RegisteredPayload` |
| `authenticated` | 认证成功 | `AuthenticatedPayload` |
| `auth_failed` | 认证失败 | `AuthFailedPayload` |
| `pong` | 心跳响应 | - |
| `run_task` | 下发任务 | `RunTaskPayload` |
| `stop_task` | 停止任务 | `StopTaskPayload` |
| `request_screenshot` | 请求截图 | `RequestScreenshotPayload` |
| `error` | 错误通知 | `ErrorPayload` |

### Payload 定义

详见 `client/protocol.go`。

## 开发指南

### 环境搭建

```bash
# 克隆代码
git clone <repository>
cd maaend-client

# 下载依赖
go mod download

# 编译
go build -o maaend-client.exe .

# 运行
./maaend-client -maaend /path/to/MaaEnd -debug
```

### 编译选项

```bash
# 标准编译
go build -o maaend-client .

# 优化编译（减小体积）
go build -ldflags="-s -w" -o maaend-client .

# 交叉编译
GOOS=windows GOARCH=amd64 go build -o maaend-client.exe .
GOOS=linux GOARCH=amd64 go build -o maaend-client-linux .
GOOS=darwin GOARCH=amd64 go build -o maaend-client-mac .
```

### 添加新消息类型

1. 在 `client/protocol.go` 中添加消息类型常量和 Payload 结构：

```go
// 新消息类型
const MsgTypeNewMessage = "new_message"

// 新消息 Payload
type NewMessagePayload struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}
```

2. 在 `client/handler.go` 中添加处理逻辑：

```go
func (c *Client) handleMessage(msg *Message) {
    switch msg.Type {
    // ... 现有处理
    case MsgTypeNewMessage:
        c.handleNewMessage(msg)
    }
}

func (c *Client) handleNewMessage(msg *Message) {
    var payload NewMessagePayload
    if err := msg.ParsePayload(&payload); err != nil {
        log.Printf("[Client] 解析 NewMessage 失败: %v", err)
        return
    }
    // 处理逻辑
}
```

### 添加新控制器类型

1. 在 `maa/wrapper.go` 的 `ConnectController` 中添加 case：

```go
switch ctrlConfig.Type {
case "Win32":
    ctrl, err = w.createWin32Controller(ctrlConfig)
case "Adb":
    ctrl, err = w.createAdbController(ctrlConfig)
case "NewType":
    ctrl, err = w.createNewTypeController(ctrlConfig)
}
```

2. 实现 `createNewTypeController` 方法。

### 调试技巧

1. **启用调试模式**：
```bash
./maaend-client -debug
```

2. **查看 MaaFramework 日志**：
日志输出到 `MaaEnd/debug/` 目录。

3. **WebSocket 消息调试**：
在 `client/client.go` 的 `Send` 方法中添加日志。

4. **控制器连接调试**：
在 `maa/wrapper.go` 的 `createWin32Controller` 中已有详细日志。

## 注意事项

### 并发安全

- `Client.connected` 使用 `sync.RWMutex` 保护
- `Client.currentJob` 使用 `sync.Mutex` 保护
- `Wrapper` 的方法使用 `sync.Mutex` 保护

### Channel 关闭

任务完成后，需要先调用 `ClearEventChannels()` 清除事件通道引用，再关闭 channel，防止 panic。

```go
// 正确做法
c.maaWrapper.ClearEventChannels()
close(statusCh)
close(logCh)

// 错误做法 - 可能导致 panic
close(statusCh)  // wrapper 可能还在写入
```

### 资源释放

确保在退出时正确释放资源：

```go
defer maaWrapper.Cleanup()  // 释放 MaaFramework 资源
defer wsClient.Stop()       // 关闭 WebSocket 连接
```

### 配置文件格式

使用 `SaveConfig()` 而非 `viper.WriteConfig()` 保存配置，以保持格式整洁和注释。

## API 参考

### MaaWrapperInterface

```go
type MaaWrapperInterface interface {
    // 获取设备能力（任务列表、控制器、资源）
    GetCapabilities() (*CapabilitiesPayload, error)
    
    // 执行任务
    RunTask(job *Job, statusCh chan<- TaskStatusPayload, logCh chan<- TaskLogPayload) error
    
    // 停止任务
    StopTask() error
    
    // 截图
    TakeScreenshot() ([]byte, int, int, error)
    
    // 清除事件通道引用
    ClearEventChannels()
    
    // 获取 MaaEnd 版本
    GetVersion() string
}
```

### Client 公开方法

```go
// 创建客户端
func NewClient(cfg *config.Config) *Client

// 设置 MaaWrapper
func (c *Client) SetMaaWrapper(wrapper MaaWrapperInterface)

// 设置回调
func (c *Client) SetCallbacks(onConnected, onDisconnected func(), onMessage func(*Message))

// 运行客户端（阻塞）
func (c *Client) Run(ctx context.Context) error

// 停止客户端
func (c *Client) Stop()

// 检查是否已连接
func (c *Client) IsConnected() bool

// 检查是否有 Token
func (c *Client) HasToken() bool

// 发送注册消息
func (c *Client) SendRegister(bindCode string)

// 发送能力上报
func (c *Client) SendCapabilities()
```

## 版本历史

### v0.2.0 (2026-01-31)

- 初始版本
- WebSocket 连接和认证
- 任务执行和状态上报
- 实时截图
- 自动重连
- 多控制器支持（Win32/ADB）

## 贡献指南

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

MIT License
