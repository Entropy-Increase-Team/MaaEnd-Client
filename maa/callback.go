package maa

import (
	"fmt"
	"log"
	"sync"

	maafw "github.com/MaaXYZ/maa-framework-go/v3"

	"maaend-client/client"
)

// EventHandler 事件处理器
type EventHandler struct {
	statusCh chan<- client.TaskStatusPayload
	logCh    chan<- client.TaskLogPayload
	jobID    string
	mu       sync.RWMutex
}

// NewEventHandler 创建事件处理器
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// SetChannels 设置通道
func (h *EventHandler) SetChannels(statusCh chan<- client.TaskStatusPayload, logCh chan<- client.TaskLogPayload, jobID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.statusCh = statusCh
	h.logCh = logCh
	h.jobID = jobID
}

// ClearChannels 清除通道引用（在关闭通道前调用，防止 panic）
func (h *EventHandler) ClearChannels() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.statusCh = nil
	h.logCh = nil
	h.jobID = ""
}

// SendStatus 安全发送状态（带 panic 保护）
func (h *EventHandler) SendStatus(payload client.TaskStatusPayload) bool {
	h.mu.RLock()
	ch := h.statusCh
	h.mu.RUnlock()

	if ch == nil {
		return false
	}
	return safeSendStatus(ch, payload)
}

// SendLog 安全发送日志（带 panic 保护）
func (h *EventHandler) SendLog(payload client.TaskLogPayload) bool {
	h.mu.RLock()
	ch := h.logCh
	h.mu.RUnlock()

	if ch == nil {
		return false
	}
	return safeSendLog(ch, payload)
}

// GetJobID 获取当前任务ID
func (h *EventHandler) GetJobID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.jobID
}

// OnTaskerTask 处理 Tasker 任务事件
func (h *EventHandler) OnTaskerTask(event maafw.EventStatus, detail maafw.TaskerTaskDetail) {
	h.mu.RLock()
	logCh := h.logCh
	jobID := h.jobID
	h.mu.RUnlock()

	if logCh == nil {
		return
	}

	level := "info"
	message := ""

	switch event {
	case maafw.EventStatusStarting:
		message = fmt.Sprintf("任务开始: %s", detail.Entry)
	case maafw.EventStatusSucceeded:
		message = fmt.Sprintf("任务成功: %s", detail.Entry)
	case maafw.EventStatusFailed:
		level = "error"
		message = fmt.Sprintf("任务失败: %s", detail.Entry)
	default:
		message = fmt.Sprintf("任务事件 %d: %s", event, detail.Entry)
	}

	if message != "" {
		safeSendLog(logCh, client.TaskLogPayload{
			JobID:     jobID,
			Level:     level,
			Message:   message,
			EventType: "task",
		})
	}
}

// OnNodePipelineNode 处理节点事件（备用，暂未使用）
func (h *EventHandler) OnNodePipelineNode(event maafw.EventStatus, detail maafw.NodePipelineNodeDetail) {
	h.mu.RLock()
	logCh := h.logCh
	jobID := h.jobID
	h.mu.RUnlock()

	if logCh == nil {
		return
	}

	level := "debug"
	message := ""

	switch event {
	case maafw.EventStatusStarting:
		message = fmt.Sprintf("节点开始: %s", detail.Name)
	case maafw.EventStatusSucceeded:
		message = fmt.Sprintf("节点完成: %s", detail.Name)
	case maafw.EventStatusFailed:
		level = "warn"
		message = fmt.Sprintf("节点失败: %s", detail.Name)
	}

	if message != "" {
		safeSendLog(logCh, client.TaskLogPayload{
			JobID:     jobID,
			Level:     level,
			Message:   message,
			NodeName:  detail.Name,
			EventType: "node",
		})
	}
}

// safeSendStatus 安全发送状态，防止向已关闭的 channel 写入导致 panic
func safeSendStatus(ch chan<- client.TaskStatusPayload, payload client.TaskStatusPayload) bool {
	defer func() {
		if recover() != nil {
			// 通道可能已关闭，忽略并防止崩溃
		}
	}()
	select {
	case ch <- payload:
		return true
	default:
		// 队列满，跳过
		return false
	}
}

// safeSendLog 安全发送日志，防止向已关闭的 channel 写入导致 panic
func safeSendLog(ch chan<- client.TaskLogPayload, payload client.TaskLogPayload) bool {
	defer func() {
		if recover() != nil {
			// 通道可能已关闭，忽略并防止崩溃
		}
	}()
	select {
	case ch <- payload:
		return true
	default:
		// 队列满，跳过
		return false
	}
}

// formatValue 格式化值
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func init() {
	// 初始化日志
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
}
