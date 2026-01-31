package maa

import (
	"fmt"

	"maaend-client/client"
	"maaend-client/core"
)

// TaskExecutor 任务执行器
type TaskExecutor struct {
	wrapper    *Wrapper
	resolver   *core.OptionResolver
	jobID      string
	statusCh   chan<- client.TaskStatusPayload
	logCh      chan<- client.TaskLogPayload
	totalTasks int
	completed  int
}

// NewTaskExecutor 创建任务执行器
func NewTaskExecutor(wrapper *Wrapper, jobID string, statusCh chan<- client.TaskStatusPayload, logCh chan<- client.TaskLogPayload) *TaskExecutor {
	return &TaskExecutor{
		wrapper:  wrapper,
		resolver: core.NewOptionResolver(wrapper.pi),
		jobID:    jobID,
		statusCh: statusCh,
		logCh:    logCh,
	}
}

// Execute 执行任务列表
func (e *TaskExecutor) Execute(tasks []client.RunTaskItem) error {
	e.totalTasks = len(tasks)
	e.completed = 0

	for i, task := range tasks {
		if e.wrapper.stopRequested {
			return fmt.Errorf("任务被停止")
		}

		if err := e.executeTask(task, i); err != nil {
			return err
		}

		e.completed = i + 1
	}

	return nil
}

// executeTask 执行单个任务
func (e *TaskExecutor) executeTask(task client.RunTaskItem, _ int) error {
	// 获取任务配置
	taskConfig := e.wrapper.pi.GetTask(task.Name)
	if taskConfig == nil {
		return fmt.Errorf("任务不存在: %s", task.Name)
	}

	// 发送状态
	e.sendStatus("running", task.Name, fmt.Sprintf("正在执行: %s", taskConfig.Label))

	// 解析选项
	override, err := e.resolver.ResolveTaskOptions(task.Name, task.Options)
	if err != nil {
		return fmt.Errorf("解析选项失败: %w", err)
	}

	// 发送日志
	e.sendLog("info", fmt.Sprintf("开始执行任务: %s", taskConfig.Label), task.Name)

	// 执行任务
	taskJob := e.wrapper.tasker.PostTask(taskConfig.Entry, override)
	taskJob.Wait()

	if taskJob.Failure() {
		e.sendLog("error", fmt.Sprintf("任务执行失败: %s", task.Name), task.Name)
		return fmt.Errorf("任务执行失败: %s", task.Name)
	}

	e.sendLog("info", fmt.Sprintf("任务完成: %s", taskConfig.Label), task.Name)
	return nil
}

// sendStatus 发送状态
func (e *TaskExecutor) sendStatus(status, currentTask, message string) {
	if e.statusCh == nil {
		return
	}
	e.statusCh <- client.TaskStatusPayload{
		JobID:       e.jobID,
		Status:      status,
		CurrentTask: currentTask,
		Progress: client.JobProgress{
			Completed: e.completed,
			Total:     e.totalTasks,
		},
		Message: message,
	}
}

// sendLog 发送日志
func (e *TaskExecutor) sendLog(level, message, nodeName string) {
	if e.logCh == nil {
		return
	}
	e.logCh <- client.TaskLogPayload{
		JobID:     e.jobID,
		Level:     level,
		Message:   message,
		NodeName:  nodeName,
		EventType: "task",
	}
}
