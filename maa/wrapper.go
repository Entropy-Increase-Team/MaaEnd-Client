package maa

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	maafw "github.com/MaaXYZ/maa-framework-go/v3"
	"github.com/MaaXYZ/maa-framework-go/v3/controller/win32"

	"maaend-client/client"
	"maaend-client/config"
	"maaend-client/core"
)

// Wrapper MaaFramework 封装
type Wrapper struct {
	maaEndPath string
	pi         *core.ProjectInterface

	controller *maafw.Controller
	resource   *maafw.Resource
	tasker     *maafw.Tasker

	// 当前连接的控制器和资源名称
	currentController string
	currentResource   string

	// Agent 服务
	agentServer *AgentServer

	// 事件处理
	eventHandler *EventHandler

	// 状态
	initialized bool
	mu          sync.Mutex

	// 任务控制
	stopRequested bool
}

// NewWrapper 创建 Wrapper
func NewWrapper(maaEndPath string) *Wrapper {
	return &Wrapper{
		maaEndPath:   maaEndPath,
		eventHandler: NewEventHandler(),
	}
}

// Init 初始化 MaaFramework
func (w *Wrapper) Init() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.initialized {
		return nil
	}

	log.Printf("[Maa] 初始化 MaaFramework...")

	// 加载 interface.json
	pi, err := core.LoadInterface(w.maaEndPath)
	if err != nil {
		return fmt.Errorf("加载 interface.json 失败: %w", err)
	}
	if cfg := config.Get(); cfg != nil {
		applyWin32Overrides(pi, cfg.MaaEnd.Win32ClassRegex, cfg.MaaEnd.Win32WindowRegex)
	}
	w.pi = pi
	log.Printf("[Maa] 加载项目: %s v%s", pi.Name, pi.Version)

	// 初始化日志目录
	logDir := filepath.Join(w.maaEndPath, "debug")
	os.MkdirAll(logDir, 0755)

	// 初始化 MaaFramework
	maafwPath := pi.GetMaaFWPath()
	err = maafw.Init(
		maafw.WithLibDir(maafwPath),
		maafw.WithLogDir(logDir),
		maafw.WithStdoutLevel(maafw.LoggingLevelInfo),
	)
	if err != nil && err != maafw.ErrAlreadyInitialized {
		return fmt.Errorf("初始化 MaaFramework 失败: %w", err)
	}

	w.initialized = true
	log.Printf("[Maa] MaaFramework 初始化完成")

	return nil
}

func applyWin32Overrides(pi *core.ProjectInterface, classRegex, windowRegex string) {
	if pi == nil {
		return
	}
	if classRegex == "" && windowRegex == "" {
		return
	}
	for i := range pi.Controllers {
		ctrl := &pi.Controllers[i]
		if ctrl.Win32 == nil {
			continue
		}
		if classRegex != "" {
			ctrl.Win32.ClassRegex = classRegex
		}
		if windowRegex != "" {
			ctrl.Win32.WindowRegex = windowRegex
		}
	}
	log.Printf("[Maa] 已覆盖 Win32 窗口匹配规则: class=%q, window=%q", classRegex, windowRegex)
}

// GetCapabilities 获取设备能力
func (w *Wrapper) GetCapabilities() (*client.CapabilitiesPayload, error) {
	if !w.initialized {
		return nil, fmt.Errorf("MaaFramework 未初始化")
	}

	builder := core.NewCapabilitiesBuilder(w.pi, "zh_cn")
	return builder.Build(), nil
}

// ConnectController 连接控制器
func (w *Wrapper) ConnectController(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentController == name && w.controller != nil {
		return nil // 已连接
	}

	// 获取控制器配置
	ctrlConfig := w.pi.GetController(name)
	if ctrlConfig == nil {
		return fmt.Errorf("控制器不存在: %s", name)
	}

	log.Printf("[Maa] 连接控制器: %s (类型: %s)", name, ctrlConfig.Type)

	// 关闭旧控制器
	if w.controller != nil {
		w.controller.Destroy()
		w.controller = nil
	}

	// 创建新控制器
	var ctrl *maafw.Controller
	var err error

	switch ctrlConfig.Type {
	case "Win32":
		ctrl, err = w.createWin32Controller(ctrlConfig)
	case "Adb":
		ctrl, err = w.createAdbController(ctrlConfig)
	default:
		return fmt.Errorf("不支持的控制器类型: %s", ctrlConfig.Type)
	}

	if err != nil {
		return fmt.Errorf("创建控制器失败: %w", err)
	}

	if ctrl == nil {
		return fmt.Errorf("创建控制器失败: 返回 nil")
	}

	// 等待连接
	ctrl.PostConnect().Wait()

	if !ctrl.Connected() {
		ctrl.Destroy()
		return fmt.Errorf("控制器连接失败")
	}

	w.controller = ctrl
	w.currentController = name

	log.Printf("[Maa] 控制器连接成功: %s", name)
	return nil
}

// createWin32Controller 创建 Win32 控制器
func (w *Wrapper) createWin32Controller(config *core.ControllerConfig) (*maafw.Controller, error) {
	if config.Win32 == nil {
		return nil, fmt.Errorf("Win32 配置缺失")
	}

	// 打印控制器配置详情
	log.Printf("[Maa] Win32 控制器配置:")
	log.Printf("[Maa]   class_regex: %s", config.Win32.ClassRegex)
	log.Printf("[Maa]   window_regex: %s", config.Win32.WindowRegex)
	log.Printf("[Maa]   screencap: %s", config.Win32.Screencap)
	log.Printf("[Maa]   mouse: %s", config.Win32.Mouse)
	log.Printf("[Maa]   keyboard: %s", config.Win32.Keyboard)

	// 查找窗口
	windows := maafw.FindDesktopWindows()
	if len(windows) == 0 {
		return nil, fmt.Errorf("未找到窗口")
	}

	// 匹配窗口
	var targetWindow *maafw.DesktopWindow
	for _, win := range windows {
		if matchWindow(win, config.Win32.ClassRegex, config.Win32.WindowRegex) {
			targetWindow = win
			log.Printf("[Maa] 匹配到窗口: handle=%v, class=%s, name=%s",
				win.Handle, win.ClassName, win.WindowName)
			break
		}
	}

	if targetWindow == nil {
		return nil, fmt.Errorf("未找到匹配的窗口 (class: %s, window: %s)",
			config.Win32.ClassRegex, config.Win32.WindowRegex)
	}

	// 解析方法
	screencapMethod := parseScreencapMethod(config.Win32.Screencap)
	mouseMethod := parseInputMethod(config.Win32.Mouse)
	keyboardMethod := parseInputMethod(config.Win32.Keyboard)

	log.Printf("[Maa] 解析后的方法: screencap=%v, mouse=%v, keyboard=%v",
		screencapMethod, mouseMethod, keyboardMethod)

	// 创建控制器
	ctrl := maafw.NewWin32Controller(
		targetWindow.Handle,
		screencapMethod,
		mouseMethod,
		keyboardMethod,
	)

	return ctrl, nil
}

// createAdbController 创建 ADB 控制器
func (w *Wrapper) createAdbController(_ *core.ControllerConfig) (*maafw.Controller, error) {
	// 查找设备
	devices := maafw.FindAdbDevices()
	if len(devices) == 0 {
		return nil, fmt.Errorf("未找到 ADB 设备")
	}

	// 使用第一个设备
	device := devices[0]

	ctrl := maafw.NewAdbController(
		device.AdbPath,
		device.Address,
		device.ScreencapMethod,
		device.InputMethod,
		device.Config,
		"",
	)

	return ctrl, nil
}

// LoadResource 加载资源
func (w *Wrapper) LoadResource(name string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentResource == name && w.resource != nil {
		return nil // 已加载
	}

	log.Printf("[Maa] 加载资源: %s", name)

	// 获取资源路径
	paths := w.pi.GetResourcePaths(name)
	if len(paths) == 0 {
		return fmt.Errorf("资源不存在: %s", name)
	}

	// 释放旧资源
	if w.resource != nil {
		w.resource.Destroy()
		w.resource = nil
	}

	// 创建资源
	res := maafw.NewResource()
	if res == nil {
		return fmt.Errorf("创建资源失败")
	}

	// 加载每个路径
	for _, path := range paths {
		log.Printf("[Maa] 加载资源路径: %s", path)
		res.PostBundle(path).Wait()
	}

	w.resource = res
	w.currentResource = name

	log.Printf("[Maa] 资源加载完成: %s", name)
	return nil
}

// RunTask 执行任务
func (w *Wrapper) RunTask(job *client.Job, statusCh chan<- client.TaskStatusPayload, logCh chan<- client.TaskLogPayload) error {
	w.mu.Lock()
	if !w.initialized {
		w.mu.Unlock()
		return fmt.Errorf("MaaFramework 未初始化")
	}
	w.stopRequested = false
	w.mu.Unlock()

	// 连接控制器
	if err := w.ConnectController(job.Controller); err != nil {
		return fmt.Errorf("连接控制器失败: %w", err)
	}

	// 加载资源
	if err := w.LoadResource(job.Resource); err != nil {
		return fmt.Errorf("加载资源失败: %w", err)
	}

	// 创建 Tasker
	if w.tasker != nil {
		w.tasker.Destroy()
	}
	w.tasker = maafw.NewTasker()
	if w.tasker == nil {
		return fmt.Errorf("创建 Tasker 失败")
	}

	// 绑定控制器和资源
	w.tasker.BindController(w.controller)
	w.tasker.BindResource(w.resource)

	if !w.tasker.Initialized() {
		return fmt.Errorf("Tasker 初始化失败")
	}

	// 注册事件回调
	w.eventHandler.SetChannels(statusCh, logCh, job.JobID)
	w.tasker.OnTaskerTask(func(status maafw.EventStatus, detail maafw.TaskerTaskDetail) {
		w.eventHandler.OnTaskerTask(status, detail)
	})

	// 启动 Agent（如果配置了）
	if w.pi.GetAgentExec() != "" {
		if err := w.startAgent(); err != nil {
			log.Printf("[Maa] 启动 Agent 失败: %v (继续执行)", err)
		}
	}

	// 创建选项解析器
	resolver := core.NewOptionResolver(w.pi)

	// 执行每个任务
	total := len(job.Tasks)
	for i, taskItem := range job.Tasks {
		if w.stopRequested {
			return fmt.Errorf("任务被停止")
		}

		// 获取任务配置
		taskConfig := w.pi.GetTask(taskItem.Name)
		if taskConfig == nil {
			log.Printf("[Maa] 任务不存在: %s", taskItem.Name)
			continue
		}

		log.Printf("[Maa] 执行任务 [%d/%d]: %s", i+1, total, taskItem.Name)

		// 发送状态（使用安全方法防止 channel 已关闭）
		w.eventHandler.SendStatus(client.TaskStatusPayload{
			JobID:       job.JobID,
			Status:      "running",
			CurrentTask: taskItem.Name,
			Progress:    client.JobProgress{Completed: i, Total: total},
			Message:     fmt.Sprintf("正在执行: %s", taskConfig.Label),
		})

		// 解析选项
		override, err := resolver.ResolveTaskOptions(taskItem.Name, taskItem.Options)
		if err != nil {
			return fmt.Errorf("解析选项失败: %w", err)
		}

		// 执行任务
		taskJob := w.tasker.PostTask(taskConfig.Entry, override)
		taskJob.Wait()

		if taskJob.Failure() {
			return fmt.Errorf("任务执行失败: %s", taskItem.Name)
		}

		log.Printf("[Maa] 任务完成: %s", taskItem.Name)
	}

	return nil
}

// StopTask 停止任务
func (w *Wrapper) StopTask() error {
	w.mu.Lock()
	w.stopRequested = true
	w.mu.Unlock()

	if w.tasker != nil {
		w.tasker.PostStop()
	}

	log.Printf("[Maa] 任务停止请求已发送")
	return nil
}

// ClearEventChannels 清除事件通道引用（在关闭通道前调用，防止 panic）
func (w *Wrapper) ClearEventChannels() {
	if w.eventHandler != nil {
		w.eventHandler.ClearChannels()
	}
}

// TakeScreenshot 截图
func (w *Wrapper) TakeScreenshot() ([]byte, int, int, error) {
	if w.controller == nil {
		return nil, 0, 0, fmt.Errorf("控制器未连接")
	}

	// 获取截图
	w.controller.PostScreencap().Wait()
	img := w.controller.CacheImage()
	if img == nil {
		return nil, 0, 0, fmt.Errorf("截图失败")
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, fmt.Errorf("编码截图失败: %w", err)
	}

	bounds := img.Bounds()
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), nil
}

// startAgent 启动 Agent
func (w *Wrapper) startAgent() error {
	agentExec := w.pi.GetAgentExec()
	if agentExec == "" {
		return nil
	}

	// 检查 Agent 可执行文件是否存在
	if _, err := os.Stat(agentExec); os.IsNotExist(err) {
		return fmt.Errorf("Agent 服务文件不存在: %s (部分功能如自定义识别器将不可用)", agentExec)
	}

	if w.agentServer == nil {
		w.agentServer = NewAgentServer()
	}

	return w.agentServer.Start(agentExec, w.pi.Agent.ChildArgs)
}

// Cleanup 清理资源
func (w *Wrapper) Cleanup() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.agentServer != nil {
		w.agentServer.Stop()
		w.agentServer = nil
	}

	if w.tasker != nil {
		w.tasker.Destroy()
		w.tasker = nil
	}

	if w.resource != nil {
		w.resource.Destroy()
		w.resource = nil
	}

	if w.controller != nil {
		w.controller.Destroy()
		w.controller = nil
	}

	w.initialized = false
	log.Printf("[Maa] 资源已清理")
}

// GetProjectInterface 获取项目接口
func (w *Wrapper) GetProjectInterface() *core.ProjectInterface {
	return w.pi
}

// GetVersion 获取 MaaEnd 版本
func (w *Wrapper) GetVersion() string {
	if w.pi == nil {
		return "unknown"
	}
	if w.pi.Version == "" {
		return "unknown"
	}
	return w.pi.Version
}

// matchWindow 使用正则表达式匹配窗口
func matchWindow(win *maafw.DesktopWindow, classRegex, windowRegex string) bool {
	// 匹配 class 名称
	if classRegex != "" {
		if win.ClassName == "" {
			return false
		}
		matched, err := regexp.MatchString(classRegex, win.ClassName)
		if err != nil {
			// 正则表达式语法错误，回退到包含匹配
			log.Printf("[Maa] class_regex 语法错误 '%s': %v, 回退到包含匹配", classRegex, err)
			if !containsString(win.ClassName, classRegex) {
				return false
			}
		} else if !matched {
			return false
		}
	}

	// 匹配窗口名称
	if windowRegex != "" {
		if win.WindowName == "" {
			return false
		}
		matched, err := regexp.MatchString(windowRegex, win.WindowName)
		if err != nil {
			// 正则表达式语法错误，回退到包含匹配
			log.Printf("[Maa] window_regex 语法错误 '%s': %v, 回退到包含匹配", windowRegex, err)
			if !containsString(win.WindowName, windowRegex) {
				return false
			}
		} else if !matched {
			return false
		}
	}

	return true
}

// containsString 检查字符串是否包含子串（用于正则表达式语法错误时的回退）
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// parseScreencapMethod 解析截图方法
// 支持的方法（来自 maa-framework-go/v3/controller/win32）：
//   - ScreencapGDI (1)
//   - ScreencapFramePool (2)
//   - ScreencapDXGIDesktopDup (4)
//   - ScreencapDXGIDesktopDupWindow (8)
//   - ScreencapPrintWindow (16)
//   - ScreencapScreenDC (32)
func parseScreencapMethod(method string) win32.ScreencapMethod {
	// 首先尝试使用官方解析
	m, err := win32.ParseScreencapMethod(method)
	if err == nil {
		return m
	}

	// 手动映射常见的截图方法（与 interface.json 中的命名一致）
	switch method {
	case "GDI":
		return win32.ScreencapGDI
	case "FramePool":
		return win32.ScreencapFramePool
	case "DXGI_DesktopDup":
		return win32.ScreencapDXGIDesktopDup
	case "DXGI_DesktopDup_Window":
		return win32.ScreencapDXGIDesktopDupWindow
	case "PrintWindow":
		return win32.ScreencapPrintWindow
	case "ScreenDC":
		return win32.ScreencapScreenDC
	default:
		log.Printf("[Maa] 未知截图方法 '%s', 使用默认 FramePool", method)
		return win32.ScreencapFramePool
	}
}

// parseInputMethod 解析输入方法
// 支持的方法（来自 maa-framework-go/v3/controller/win32）：
//   - InputSeize (1) - 前台模式，独占输入
//   - InputSendMessage (2)
//   - InputPostMessage (4)
//   - InputLegacyEvent (8)
//   - InputPostThreadMessage (16)
//   - InputSendMessageWithCursorPos (32) - 带光标位置的 SendMessage
//   - InputPostMessageWithCursorPos (64)
//   - InputSendMessageWithCursorPosAndBlockInput (128)
//   - InputPostMessageWithCursorPosAndBlockInput (256)
func parseInputMethod(method string) win32.InputMethod {
	// 首先尝试使用官方解析
	m, err := win32.ParseInputMethod(method)
	if err == nil {
		return m
	}

	// 手动映射常见的输入方法（与 interface.json 中的命名一致）
	switch method {
	case "Seize":
		return win32.InputSeize
	case "SendMessage":
		return win32.InputSendMessage
	case "PostMessage":
		return win32.InputPostMessage
	case "SendMessageWithCursorPos":
		return win32.InputSendMessageWithCursorPos
	case "PostMessageWithCursorPos":
		return win32.InputPostMessageWithCursorPos
	case "SendMessageWithCursorPosAndBlockInput":
		return win32.InputSendMessageWithCursorPosAndBlockInput
	case "PostMessageWithCursorPosAndBlockInput":
		return win32.InputPostMessageWithCursorPosAndBlockInput
	default:
		log.Printf("[Maa] 未知输入方法 '%s', 使用默认 SendMessage", method)
		return win32.InputSendMessage
	}
}
