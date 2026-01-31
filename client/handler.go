package client

import (
	"encoding/base64"
	"log"
	"time"

	"maaend-client/config"
)

// handleMessage 处理服务端消息
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MsgTypeRegistered:
		c.handleRegistered(msg)
	case MsgTypeAuthenticated:
		c.handleAuthenticated(msg)
	case MsgTypeAuthFailed:
		c.handleAuthFailed(msg)
	case MsgTypePong:
		c.handlePong(msg)
	case MsgTypeRunTask:
		c.handleRunTask(msg)
	case MsgTypeStopTask:
		c.handleStopTask(msg)
	case MsgTypeRequestScreenshot:
		c.handleRequestScreenshot(msg)
	case MsgTypeError:
		c.handleError(msg)
	default:
		log.Printf("[Client] 未知消息类型: %s", msg.Type)
	}

	// 调用外部回调
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}

// handleRegistered 处理注册成功
func (c *Client) handleRegistered(msg *Message) {
	var payload RegisteredPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析注册响应失败: %v", err)
		return
	}

	c.deviceID = payload.DeviceID
	c.deviceToken = payload.DeviceToken

	log.Printf("[Client] 注册成功！设备ID: %s", payload.DeviceID)

	// 保存 token 到配置文件
	if err := config.SaveToken(payload.DeviceToken); err != nil {
		log.Printf("[Client] 保存设备令牌失败: %v", err)
	} else {
		log.Printf("[Client] 设备令牌已保存")
	}

	// 上报设备能力
	c.SendCapabilities()
}

// handleAuthenticated 处理认证成功
func (c *Client) handleAuthenticated(msg *Message) {
	var payload AuthenticatedPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析认证响应失败: %v", err)
		return
	}

	c.deviceID = payload.DeviceID

	log.Printf("[Client] 认证成功！设备ID: %s, 用户: %s", payload.DeviceID, payload.UserNickname)

	// 上报设备能力
	c.SendCapabilities()
}

// handleAuthFailed 处理认证失败
func (c *Client) handleAuthFailed(msg *Message) {
	var payload AuthFailedPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析认证失败响应失败: %v", err)
		return
	}

	log.Printf("[Client] 认证失败: %s - %s", payload.Error, payload.Message)

	// 清除本地 token
	c.deviceToken = ""
	config.SaveToken("")

	log.Printf("[Client] 已清除本地令牌，请重新绑定设备")
}

// handlePong 处理心跳响应
func (c *Client) handlePong(_ *Message) {
	// 心跳响应，正常
}

// handleRunTask 处理任务执行请求
func (c *Client) handleRunTask(msg *Message) {
	var payload RunTaskPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析任务请求失败: %v", err)
		return
	}

	log.Printf("[Client] 收到任务: %s, 控制器: %s, 资源: %s, 任务数: %d",
		payload.JobID, payload.Controller, payload.Resource, len(payload.Tasks))

	// 检查是否有正在执行的任务
	if c.GetCurrentJob() != nil {
		log.Printf("[Client] 已有任务正在执行，拒绝新任务")
		c.SendTaskCompleted(&TaskCompletedPayload{
			JobID:      payload.JobID,
			Status:     "failed",
			Error:      "设备忙碌",
			DurationMs: 0,
		})
		return
	}

	// 检查 MaaWrapper
	if c.maaWrapper == nil {
		log.Printf("[Client] MaaWrapper 未初始化")
		c.SendTaskCompleted(&TaskCompletedPayload{
			JobID:      payload.JobID,
			Status:     "failed",
			Error:      "MaaFramework 未初始化",
			DurationMs: 0,
		})
		return
	}

	// 创建任务
	job := &Job{
		JobID:      payload.JobID,
		Controller: payload.Controller,
		Resource:   payload.Resource,
		Tasks:      payload.Tasks,
		StartTime:  time.Now(),
		Status:     "running",
	}
	c.SetCurrentJob(job)

	// 发送任务开始状态
	c.SendTaskStatus(&TaskStatusPayload{
		JobID:       payload.JobID,
		Status:      "running",
		CurrentTask: "",
		Progress:    JobProgress{Completed: 0, Total: len(payload.Tasks)},
		Message:     "任务开始执行",
	})

	// 异步执行任务
	go c.executeTask(job)
}

// executeTask 执行任务
func (c *Client) executeTask(job *Job) {
	startTime := time.Now()

	// 创建状态和日志通道
	statusCh := make(chan TaskStatusPayload, 100)
	logCh := make(chan TaskLogPayload, 1000)

	// 启动状态转发协程
	go func() {
		for status := range statusCh {
			c.SendTaskStatus(&status)
		}
	}()

	// 启动日志转发协程
	go func() {
		for logEntry := range logCh {
			c.SendTaskLog(&logEntry)
		}
	}()

	// 执行任务
	err := c.maaWrapper.RunTask(job, statusCh, logCh)

	// 关闭通道
	close(statusCh)
	close(logCh)

	// 计算耗时
	duration := time.Since(startTime).Milliseconds()

	// 清除当前任务
	c.ClearCurrentJob()

	// 发送任务完成
	if err != nil {
		log.Printf("[Client] 任务执行失败: %v", err)
		c.SendTaskCompleted(&TaskCompletedPayload{
			JobID:      job.JobID,
			Status:     "failed",
			Error:      err.Error(),
			DurationMs: duration,
		})
	} else {
		log.Printf("[Client] 任务执行完成，耗时: %dms", duration)
		c.SendTaskCompleted(&TaskCompletedPayload{
			JobID:      job.JobID,
			Status:     "completed",
			DurationMs: duration,
		})
	}
}

// handleStopTask 处理停止任务请求
func (c *Client) handleStopTask(msg *Message) {
	var payload StopTaskPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析停止任务请求失败: %v", err)
		return
	}

	log.Printf("[Client] 收到停止任务请求: %s", payload.JobID)

	// 检查当前任务
	currentJob := c.GetCurrentJob()
	if currentJob == nil || currentJob.JobID != payload.JobID {
		log.Printf("[Client] 任务不存在或已完成")
		return
	}

	// 停止任务
	if c.maaWrapper != nil {
		if err := c.maaWrapper.StopTask(); err != nil {
			log.Printf("[Client] 停止任务失败: %v", err)
		}
	}

	// 任务完成回调会在 RunTask 返回后自动发送
}

// handleRequestScreenshot 处理截图请求
func (c *Client) handleRequestScreenshot(msg *Message) {
	var payload RequestScreenshotPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析截图请求失败: %v", err)
		return
	}

	log.Printf("[Client] 收到截图请求: %s", payload.RequestID)

	if c.maaWrapper == nil {
		log.Printf("[Client] MaaWrapper 未初始化，无法截图")
		return
	}

	// 异步截图
	go func() {
		imageData, width, height, err := c.maaWrapper.TakeScreenshot()
		if err != nil {
			log.Printf("[Client] 截图失败: %v", err)
			c.SendScreenshot(payload.RequestID, "", 0, 0, err.Error())
			return
		}

		// Base64 编码
		base64Image := base64.StdEncoding.EncodeToString(imageData)

		// 发送截图
		c.SendScreenshot(payload.RequestID, base64Image, width, height, "")

		log.Printf("[Client] 截图已发送: %dx%d, 大小: %d bytes",
			width, height, len(imageData))
	}()
}

// handleError 处理错误通知
func (c *Client) handleError(msg *Message) {
	var payload ErrorPayload
	if err := msg.ParsePayload(&payload); err != nil {
		log.Printf("[Client] 解析错误响应失败: %v", err)
		return
	}

	log.Printf("[Client] 服务器错误: %s - %s", payload.Code, payload.Message)
}
