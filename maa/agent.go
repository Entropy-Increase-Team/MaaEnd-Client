package maa

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

// AgentServer Agent 子进程包装。
// 按 MaaFW PI V2 约定，Client 在启动子进程时：
//   - 必须通过位置参数向子进程传入由 AgentClient 分配的 identifier / socket id；
//   - 自 v2.5.0 起，应注入 PI_* 一系列环境变量（由 Wrapper 层构造后传入）。
type AgentServer struct {
	cmd        *exec.Cmd
	identifier string
	running    bool
	mu         sync.Mutex
	exited     chan struct{}
}

// NewAgentServer 创建 Agent 子进程包装。
func NewAgentServer() *AgentServer {
	return &AgentServer{exited: make(chan struct{})}
}

// Start 启动子进程。identifier 必须非空：参考 MaaFW agent-server 示例
// （子进程会读取 os.Args[1] / len(args)>=2 作为 socket id）。
// args 为 interface.json 中声明的 child_args（已展开），identifier 会作为最后一个位置参数追加。
// env 为需要注入的额外环境变量（不会覆盖原环境，会与 os.Environ() 合并）。
func (a *AgentServer) Start(execPath string, args []string, identifier string, workDir string, env []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("agent 已在运行: %s", execPath)
	}
	if execPath == "" {
		return fmt.Errorf("agent 可执行路径为空")
	}
	if identifier == "" {
		return fmt.Errorf("agent identifier 为空，无法与 AgentClient 建立通信")
	}

	// identifier 作为最后一个位置参数追加（对齐 maa-framework-go examples/agent-server）
	fullArgs := make([]string, 0, len(args)+1)
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, identifier)

	log.Printf("[Agent] 启动: %s %v (identifier=%s)", execPath, args, identifier)

	cmd := exec.Command(execPath, fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if workDir != "" {
		cmd.Dir = workDir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 agent 子进程失败: %w", err)
	}

	a.cmd = cmd
	a.identifier = identifier
	a.running = true
	a.exited = make(chan struct{})

	go func(c *exec.Cmd, exited chan struct{}) {
		err := c.Wait()
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		close(exited)
		if err != nil {
			log.Printf("[Agent] 子进程结束: %v", err)
		} else {
			log.Printf("[Agent] 子进程正常结束")
		}
	}(cmd, a.exited)

	log.Printf("[Agent] 启动成功，PID: %d", cmd.Process.Pid)
	return nil
}

// Stop 停止子进程。
func (a *AgentServer) Stop() {
	a.mu.Lock()
	if !a.running || a.cmd == nil || a.cmd.Process == nil {
		a.mu.Unlock()
		return
	}
	cmd := a.cmd
	exited := a.exited
	a.mu.Unlock()

	log.Printf("[Agent] 停止 PID=%d...", cmd.Process.Pid)
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("[Agent] kill 子进程失败: %v", err)
	}

	// 等待子进程实际退出，避免僵尸进程（Windows 上 Wait 返回才算真正回收）
	if exited != nil {
		<-exited
	}
}

// IsRunning 检查是否运行中。
func (a *AgentServer) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// Identifier 返回启动时使用的 identifier。
func (a *AgentServer) Identifier() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.identifier
}
