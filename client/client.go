package client

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"maaend-client/config"
)

// Client WebSocket 客户端
type Client struct {
	config *config.Config
	conn   *websocket.Conn

	deviceID    string
	deviceToken string

	// 当前任务
	currentJob   *Job
	currentJobMu sync.Mutex

	// 消息通道
	sendCh chan []byte
	stopCh chan struct{}

	// 连接状态
	connected   bool
	connectedMu sync.RWMutex

	// 重连计数
	reconnectCount int

	// 回调
	onConnected    func()
	onDisconnected func()
	onMessage      func(*Message)

	// MaaWrapper 接口（外部注入）
	maaWrapper MaaWrapperInterface
}

// Job 任务信息
type Job struct {
	JobID      string
	Controller string
	Resource   string
	Tasks      []RunTaskItem
	StartTime  time.Time
	Status     string
}

// MaaWrapperInterface MaaFramework 封装接口
type MaaWrapperInterface interface {
	GetCapabilities() (*CapabilitiesPayload, error)
	RunTask(job *Job, statusCh chan<- TaskStatusPayload, logCh chan<- TaskLogPayload) error
	StopTask() error
	TakeScreenshot() ([]byte, int, int, error)
	ClearEventChannels() // 清除事件通道引用，防止关闭后写入导致 panic
	GetVersion() string  // 获取 MaaEnd 版本
}

// NewClient 创建客户端
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		sendCh: make(chan []byte, 256),
		stopCh: make(chan struct{}),
	}
}

// SetMaaWrapper 设置 MaaWrapper
func (c *Client) SetMaaWrapper(wrapper MaaWrapperInterface) {
	c.maaWrapper = wrapper
}

// SetCallbacks 设置回调
func (c *Client) SetCallbacks(onConnected, onDisconnected func(), onMessage func(*Message)) {
	c.onConnected = onConnected
	c.onDisconnected = onDisconnected
	c.onMessage = onMessage
}

// Run 运行客户端（阻塞）
func (c *Client) Run(ctx context.Context) error {
	// 加载已保存的 token
	c.deviceToken = c.config.Device.Token

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 连接服务器
		if err := c.connect(); err != nil {
			log.Printf("[Client] 连接失败: %v", err)
			c.waitForReconnect(ctx)
			continue
		}

		// 重置重连计数
		c.reconnectCount = 0

		// 连接成功回调
		if c.onConnected != nil {
			c.onConnected()
		}

		// 运行主循环
		c.runLoop(ctx)

		// 断开连接回调
		if c.onDisconnected != nil {
			c.onDisconnected()
		}

		// 检查是否应该退出
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return nil
		default:
			// 等待重连
			c.waitForReconnect(ctx)
		}
	}
}

// connect 建立 WebSocket 连接
func (c *Client) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.Server.ConnectTimeout,
	}

	header := http.Header{}
	header.Set("User-Agent", "MaaEnd-Client/1.0")

	conn, _, err := dialer.Dial(c.config.Server.WsURL, header)
	if err != nil {
		return fmt.Errorf("WebSocket 连接失败: %w", err)
	}

	c.conn = conn
	c.setConnected(true)

	log.Printf("[Client] 已连接到服务器: %s", c.config.Server.WsURL)
	return nil
}

// runLoop 主循环
func (c *Client) runLoop(ctx context.Context) {
	// 启动写协程
	go c.writeLoop(ctx)

	// 启动心跳
	go c.heartbeatLoop(ctx)

	// 认证或注册
	if c.deviceToken != "" {
		c.sendAuth()
	}

	// 读循环（阻塞）
	c.readLoop(ctx)
}

// readLoop 读消息循环
func (c *Client) readLoop(ctx context.Context) {
	defer c.close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Client] 读消息错误: %v", err)
			}
			return
		}

		// 解析消息
		msg, err := UnmarshalMessage(message)
		if err != nil {
			log.Printf("[Client] 消息解析失败: %v", err)
			continue
		}

		// 处理消息
		c.handleMessage(msg)
	}
}

// writeLoop 写消息循环
func (c *Client) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-c.sendCh:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Client] 写消息失败: %v", err)
				return
			}
		}
	}
}

// heartbeatLoop 心跳循环
func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.Server.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.isConnected() {
				return
			}
			c.sendPing()
		}
	}
}

// waitForReconnect 等待重连
func (c *Client) waitForReconnect(ctx context.Context) {
	c.reconnectCount++

	// 指数退避
	delay := time.Duration(math.Pow(2, float64(c.reconnectCount-1))) * time.Second
	maxDelay := c.config.Server.ReconnectMaxDelay
	if delay > maxDelay {
		delay = maxDelay
	}

	log.Printf("[Client] %s 后重连 (第 %d 次)", delay, c.reconnectCount)

	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
}

// close 关闭连接
func (c *Client) close() {
	c.setConnected(false)
	if c.conn != nil {
		c.conn.Close()
	}
}

// Stop 停止客户端
func (c *Client) Stop() {
	close(c.stopCh)
	c.close()
}

// ==================== 发送消息方法 ====================

// Send 发送原始消息
func (c *Client) Send(data []byte) {
	select {
	case c.sendCh <- data:
	default:
		log.Printf("[Client] 发送队列已满，丢弃消息")
	}
}

// SendMessage 发送结构化消息
func (c *Client) SendMessage(msgType string, payload interface{}) error {
	data, err := MarshalMessage(msgType, payload)
	if err != nil {
		return err
	}
	c.Send(data)
	return nil
}

// sendAuth 发送认证消息
func (c *Client) sendAuth() {
	log.Printf("[Client] 发送认证请求...")

	// 获取 MaaEnd 版本
	maaEndVersion := "unknown"
	if c.maaWrapper != nil {
		maaEndVersion = c.maaWrapper.GetVersion()
	}

	// 获取 Client 版本
	clientVersion := c.config.Version
	if clientVersion == "" {
		clientVersion = "unknown"
	}

	log.Printf("[Client] 版本信息: MaaEnd=%s, Client=%s", maaEndVersion, clientVersion)

	c.SendMessage(MsgTypeAuth, &AuthPayload{
		DeviceToken:   c.deviceToken,
		MaaEndVersion: maaEndVersion,
		ClientVersion: clientVersion,
	})
}

// SendRegister 发送注册消息
func (c *Client) SendRegister(bindCode string) {
	log.Printf("[Client] 发送注册请求，绑定码: %s", bindCode)

	// 获取 MaaEnd 版本
	maaEndVersion := "unknown"
	if c.maaWrapper != nil {
		maaEndVersion = c.maaWrapper.GetVersion()
	}

	// 获取 Client 版本
	clientVersion := c.config.Version
	if clientVersion == "" {
		clientVersion = "unknown"
	}

	log.Printf("[Client] 版本信息: MaaEnd=%s, Client=%s", maaEndVersion, clientVersion)

	c.SendMessage(MsgTypeRegister, &RegisterPayload{
		BindCode:      bindCode,
		DeviceName:    c.config.Device.Name,
		MaaEndVersion: maaEndVersion,
		ClientVersion: clientVersion,
		MaaEndPath:    c.config.MaaEnd.Path,
		OSInfo:        config.GetOSInfo(),
	})
}

// SendCapabilities 发送设备能力
func (c *Client) SendCapabilities() {
	if c.maaWrapper == nil {
		log.Printf("[Client] MaaWrapper 未初始化，跳过能力上报")
		return
	}

	capabilities, err := c.maaWrapper.GetCapabilities()
	if err != nil {
		log.Printf("[Client] 获取设备能力失败: %v", err)
		return
	}

	log.Printf("[Client] 上报设备能力: %d 个任务, %d 个控制器",
		len(capabilities.Tasks), len(capabilities.Controllers))

	c.SendMessage(MsgTypeCapabilities, capabilities)
}

// sendPing 发送心跳
func (c *Client) sendPing() {
	c.SendMessage(MsgTypePing, nil)
}

// SendTaskStatus 发送任务状态
func (c *Client) SendTaskStatus(payload *TaskStatusPayload) {
	c.SendMessage(MsgTypeTaskStatus, payload)
}

// SendTaskLog 发送任务日志
func (c *Client) SendTaskLog(payload *TaskLogPayload) {
	c.SendMessage(MsgTypeTaskLog, payload)
}

// SendTaskCompleted 发送任务完成
func (c *Client) SendTaskCompleted(payload *TaskCompletedPayload) {
	c.SendMessage(MsgTypeTaskCompleted, payload)
}

// SendScreenshot 发送截图
func (c *Client) SendScreenshot(requestID, base64Image string, width, height int, errMsg string) {
	c.SendMessage(MsgTypeScreenshot, &ScreenshotPayload{
		RequestID:   requestID,
		Base64Image: base64Image,
		Width:       width,
		Height:      height,
		Error:       errMsg,
	})
}

// ==================== 状态方法 ====================

// isConnected 检查是否已连接
func (c *Client) isConnected() bool {
	c.connectedMu.RLock()
	defer c.connectedMu.RUnlock()
	return c.connected
}

// setConnected 设置连接状态
func (c *Client) setConnected(connected bool) {
	c.connectedMu.Lock()
	c.connected = connected
	c.connectedMu.Unlock()
}

// IsConnected 公开方法：检查是否已连接
func (c *Client) IsConnected() bool {
	return c.isConnected()
}

// GetDeviceID 获取设备ID
func (c *Client) GetDeviceID() string {
	return c.deviceID
}

// HasToken 检查是否有已保存的 token
func (c *Client) HasToken() bool {
	return c.deviceToken != ""
}

// SetCurrentJob 设置当前任务
func (c *Client) SetCurrentJob(job *Job) {
	c.currentJobMu.Lock()
	c.currentJob = job
	c.currentJobMu.Unlock()
}

// GetCurrentJob 获取当前任务
func (c *Client) GetCurrentJob() *Job {
	c.currentJobMu.Lock()
	defer c.currentJobMu.Unlock()
	return c.currentJob
}

// ClearCurrentJob 清除当前任务
func (c *Client) ClearCurrentJob() {
	c.currentJobMu.Lock()
	c.currentJob = nil
	c.currentJobMu.Unlock()
}
