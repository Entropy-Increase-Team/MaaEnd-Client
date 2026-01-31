package client

import (
	"encoding/json"
	"time"
)

// ==================== 消息类型常量 ====================

// Client -> Server 消息类型
const (
	MsgTypeRegister      = "register"       // 设备注册
	MsgTypeAuth          = "auth"           // 设备认证
	MsgTypePing          = "ping"           // 心跳
	MsgTypeCapabilities  = "capabilities"   // 设备能力上报
	MsgTypeTaskStatus    = "task_status"    // 任务状态上报
	MsgTypeTaskLog       = "task_log"       // 任务日志上报
	MsgTypeTaskCompleted = "task_completed" // 任务完成上报
	MsgTypeScreenshot    = "screenshot"     // 截图上报
)

// Server -> Client 消息类型
const (
	MsgTypeRegistered        = "registered"         // 注册成功
	MsgTypeAuthenticated     = "authenticated"      // 认证成功
	MsgTypeAuthFailed        = "auth_failed"        // 认证失败
	MsgTypePong              = "pong"               // 心跳响应
	MsgTypeRunTask           = "run_task"           // 下发任务
	MsgTypeStopTask          = "stop_task"          // 停止任务
	MsgTypeRequestScreenshot = "request_screenshot" // 请求截图
	MsgTypeError             = "error"              // 错误通知
)

// ==================== 基础消息结构 ====================

// Message 基础消息结构
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewMessage 创建新消息
func NewMessage(msgType string, payload interface{}) (*Message, error) {
	var payloadBytes json.RawMessage
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	return &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}, nil
}

// ParsePayload 解析消息负载
func (m *Message) ParsePayload(v interface{}) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// ==================== Client -> Server 消息负载 ====================

// RegisterPayload 设备注册负载
type RegisterPayload struct {
	BindCode      string `json:"bind_code"`
	DeviceName    string `json:"device_name"`
	MaaEndVersion string `json:"maaend_version"` // MaaEnd 版本（来自 interface.json）
	ClientVersion string `json:"client_version"` // Client 版本（来自配置文件）
	MaaEndPath    string `json:"maaend_path"`
	OSInfo        string `json:"os_info"`
}

// AuthPayload 设备认证负载
type AuthPayload struct {
	DeviceToken   string `json:"device_token"`
	MaaEndVersion string `json:"maaend_version,omitempty"` // MaaEnd 版本
	ClientVersion string `json:"client_version,omitempty"` // Client 版本
}

// CapabilitiesPayload 设备能力上报负载
type CapabilitiesPayload struct {
	Tasks       []TaskInfo `json:"tasks"`
	Controllers []string   `json:"controllers"`
	Resources   []string   `json:"resources"`
}

// TaskInfo 任务信息
type TaskInfo struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Options     []OptionInfo `json:"options,omitempty"`
}

// OptionInfo 选项信息
type OptionInfo struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Label       string     `json:"label"`
	Cases       []CaseInfo `json:"cases,omitempty"`
	DefaultCase string     `json:"default_case,omitempty"`
}

// CaseInfo 选项分支
type CaseInfo struct {
	Name  string `json:"name"`
	Label string `json:"label"`
}

// TaskStatusPayload 任务状态上报负载
type TaskStatusPayload struct {
	JobID       string      `json:"job_id"`
	Status      string      `json:"status"`
	CurrentTask string      `json:"current_task"`
	Progress    JobProgress `json:"progress"`
	Message     string      `json:"message,omitempty"`
}

// JobProgress 任务进度
type JobProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

// TaskLogPayload 任务日志上报负载
type TaskLogPayload struct {
	JobID     string `json:"job_id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	NodeName  string `json:"node_name,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

// TaskCompletedPayload 任务完成上报负载
type TaskCompletedPayload struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// ScreenshotPayload 截图上报负载
type ScreenshotPayload struct {
	RequestID   string `json:"request_id"`
	Base64Image string `json:"base64_image"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Error       string `json:"error,omitempty"`
}

// ==================== Server -> Client 消息负载 ====================

// RegisteredPayload 注册成功响应负载
type RegisteredPayload struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token"`
}

// AuthenticatedPayload 认证成功响应负载
type AuthenticatedPayload struct {
	DeviceID     string `json:"device_id"`
	UserNickname string `json:"user_nickname"`
}

// AuthFailedPayload 认证失败响应负载
type AuthFailedPayload struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RunTaskPayload 下发任务负载
type RunTaskPayload struct {
	JobID      string        `json:"job_id"`
	Controller string        `json:"controller"`
	Resource   string        `json:"resource"`
	Tasks      []RunTaskItem `json:"tasks"`
}

// RunTaskItem 任务项
type RunTaskItem struct {
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

// StopTaskPayload 停止任务负载
type StopTaskPayload struct {
	JobID string `json:"job_id"`
}

// RequestScreenshotPayload 请求截图负载
type RequestScreenshotPayload struct {
	RequestID string `json:"request_id"`
}

// ErrorPayload 错误通知负载
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ==================== 辅助函数 ====================

// MarshalMessage 序列化消息
func MarshalMessage(msgType string, payload interface{}) ([]byte, error) {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(msg)
}

// UnmarshalMessage 反序列化消息
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
