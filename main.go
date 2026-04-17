package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"maaend-client/client"
	"maaend-client/config"
	"maaend-client/maa"
	"maaend-client/store"
)

var (
	configPath = flag.String("c", "", "配置文件路径")
	maaEndPath = flag.String("maaend", "", "MaaEnd 安装路径")
	serverURL  = flag.String("server", "", "服务器 WebSocket 地址")
	bindCode   = flag.String("bind", "", "绑定码（首次绑定时使用）")
	debugMode  = flag.Bool("debug", false, "调试模式")
)

// waitExit prints a message and waits for Enter before exiting.
// On Windows this prevents the console from closing immediately after a fatal error.
func waitExit(code int) {
	if runtime.GOOS == "windows" {
		fmt.Println("\n按 Enter 键退出...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(code)
}

// fatal logs an error and calls waitExit(1) so the user can read the message.
func fatal(format string, args ...interface{}) {
	log.Printf("[FATAL] "+format, args...)
	waitExit(1)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] 程序崩溃: %v", r)
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Printf("[PANIC] Stack trace:\n%s", buf[:n])
			waitExit(1)
		}
	}()

	flag.Parse()

	if err := ensureAdmin(); err != nil {
		fatal("需要管理员权限启动: %v", err)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if *debugMode {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	fmt.Println("========================================")
	fmt.Println("     MaaEnd Client - 远程控制客户端     ")
	fmt.Println("========================================")
	fmt.Println()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal("加载配置失败: %v", err)
	}

	if *maaEndPath != "" {
		cfg.MaaEnd.Path = *maaEndPath
	}
	if *serverURL != "" {
		cfg.Server.WsURL = *serverURL
	}

	if err := config.EnsureConfigFormat(); err != nil {
		log.Printf("警告: 无法修复配置文件格式: %v", err)
	}

	if cfg.MaaEnd.Path == "" {
		fatal("未找到 MaaEnd 安装目录，请使用 -maaend 参数指定或在 config.yaml 中配置 maaend.path")
	}
	log.Printf("MaaEnd 路径: %s", cfg.MaaEnd.Path)

	// Pre-flight checks before calling into native code
	interfacePath := filepath.Join(cfg.MaaEnd.Path, "interface.json")
	if _, err := os.Stat(interfacePath); err != nil {
		fatal("MaaEnd 路径无效: 未找到 interface.json (%s)\n请检查 maaend.path 配置是否正确", interfacePath)
	}
	maafwDir := filepath.Join(cfg.MaaEnd.Path, "maafw")
	if info, err := os.Stat(maafwDir); err != nil || !info.IsDir() {
		fatal("MaaEnd 路径无效: 未找到 maafw 目录 (%s)\n请确认 MaaEnd 已正确安装", maafwDir)
	}

	localStorage := store.NewStore("")
	if localStorage.HasCredentials() {
		cfg.Device.Token = localStorage.GetDeviceToken()
		log.Printf("已加载保存的设备凭证")
	}

	log.Printf("正在初始化 MaaFramework...")
	maaWrapper := maa.NewWrapper(cfg.MaaEnd.Path)
	maaWrapper.SetClientVersion(cfg.Version)
	if err := maaWrapper.Init(); err != nil {
		fatal("初始化 MaaFramework 失败: %v\n可能原因:\n  1. MaaEnd 路径不正确\n  2. maafw 原生库与当前系统不兼容\n  3. 缺少 Visual C++ 运行库", err)
	}
	defer maaWrapper.Cleanup()
	log.Printf("MaaFramework 初始化成功")

	wsClient := client.NewClient(cfg)
	wsClient.SetMaaWrapper(&MaaWrapperAdapter{wrapper: maaWrapper})

	wsClient.SetCallbacks(
		func() {
			log.Println("[Main] 已连接到服务器")
		},
		func() {
			log.Println("[Main] 与服务器断开连接")
		},
		func(msg *client.Message) {
			if msg.Type == client.MsgTypeRegistered {
				var payload client.RegisteredPayload
				if err := msg.ParsePayload(&payload); err == nil {
					localStorage.SaveCredentials(payload.DeviceID, payload.DeviceToken, cfg.Device.Name)
				}
			}
			if msg.Type == client.MsgTypeAuthFailed {
				localStorage.ClearCredentials()
			}
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\n收到退出信号，正在关闭...")
		cancel()
		wsClient.Stop()
	}()

	bindCodeCh := make(chan string, 1)

	if *bindCode != "" {
		bindCodeCh <- *bindCode
	} else if !wsClient.HasToken() {
		fmt.Println("\n设备未绑定，请按以下步骤操作：")
		fmt.Println("1. 在 Web 端获取绑定码")
		fmt.Println("2. 输入绑定码后按回车")
		fmt.Print("\n请输入绑定码: ")

		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				code, err := reader.ReadString('\n')
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						continue
					}
				}
				code = strings.TrimSpace(code)
				if code != "" {
					bindCodeCh <- code
					return
				}
			}
		}()
	}

	go func() {
		select {
		case code := <-bindCodeCh:
			select {
			case <-wsClient.ConnectedCh():
				wsClient.SendRegister(code)
			case <-ctx.Done():
			}
		case <-ctx.Done():
		}
	}()

	log.Printf("连接服务器: %s", cfg.Server.WsURL)
	if err := wsClient.Run(ctx); err != nil {
		if err != context.Canceled {
			log.Printf("客户端退出: %v", err)
		}
	}

	log.Println("MaaEnd Client 已退出")
}

// MaaWrapperAdapter 适配器，实现 MaaWrapperInterface
type MaaWrapperAdapter struct {
	wrapper *maa.Wrapper
}

func (a *MaaWrapperAdapter) GetCapabilities() (*client.CapabilitiesPayload, error) {
	return a.wrapper.GetCapabilities()
}

func (a *MaaWrapperAdapter) RunTask(job *client.Job, statusCh chan<- client.TaskStatusPayload, logCh chan<- client.TaskLogPayload) error {
	return a.wrapper.RunTask(job, statusCh, logCh)
}

func (a *MaaWrapperAdapter) StopTask() error {
	return a.wrapper.StopTask()
}

func (a *MaaWrapperAdapter) TakeScreenshot() ([]byte, int, int, error) {
	return a.wrapper.TakeScreenshot()
}

func (a *MaaWrapperAdapter) ClearEventChannels() {
	a.wrapper.ClearEventChannels()
}

func (a *MaaWrapperAdapter) GetVersion() string {
	return a.wrapper.GetVersion()
}
