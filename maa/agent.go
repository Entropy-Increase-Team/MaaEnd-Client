package maa

import (
	"log"
	"os"
	"os/exec"
	"sync"
)

// AgentServer Agent 服务管理
type AgentServer struct {
	cmd     *exec.Cmd
	running bool
	mu      sync.Mutex
}

// NewAgentServer 创建 Agent 服务
func NewAgentServer() *AgentServer {
	return &AgentServer{}
}

// Start 启动 Agent
func (a *AgentServer) Start(execPath string, args []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	log.Printf("[Agent] 启动: %s %v", execPath, args)

	// 检查文件是否存在
	if _, err := os.Stat(execPath); err != nil {
		return err
	}

	// 创建命令
	a.cmd = exec.Command(execPath, args...)
	a.cmd.Stdout = os.Stdout
	a.cmd.Stderr = os.Stderr

	// 启动
	if err := a.cmd.Start(); err != nil {
		return err
	}

	a.running = true

	// 等待进程结束（异步）
	go func() {
		a.cmd.Wait()
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		log.Printf("[Agent] 进程已结束")
	}()

	log.Printf("[Agent] 启动成功，PID: %d", a.cmd.Process.Pid)
	return nil
}

// Stop 停止 Agent
func (a *AgentServer) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running || a.cmd == nil || a.cmd.Process == nil {
		return
	}

	log.Printf("[Agent] 停止...")

	// 发送终止信号
	a.cmd.Process.Kill()
	a.running = false
}

// IsRunning 检查是否运行中
func (a *AgentServer) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}
