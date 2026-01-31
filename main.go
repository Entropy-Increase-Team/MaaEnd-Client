package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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

func main() {
	flag.Parse()

	// 设置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if *debugMode {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	}

	fmt.Println("========================================")
	fmt.Println("     MaaEnd Client - 远程控制客户端     ")
	fmt.Println("========================================")
	fmt.Println()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 命令行参数覆盖配置
	if *maaEndPath != "" {
		cfg.MaaEnd.Path = *maaEndPath
	}
	if *serverURL != "" {
		cfg.Server.WsURL = *serverURL
	}

	// 检查 MaaEnd 路径
	if cfg.MaaEnd.Path == "" {
		log.Fatal("未找到 MaaEnd 安装目录，请使用 -maaend 参数指定")
	}
	log.Printf("MaaEnd 路径: %s", cfg.MaaEnd.Path)

	// 初始化本地存储
	localStorage := store.NewStore("")
	if localStorage.HasCredentials() {
		cfg.Device.Token = localStorage.GetDeviceToken()
		log.Printf("已加载保存的设备凭证")
	}

	// 初始化 MaaFramework
	maaWrapper := maa.NewWrapper(cfg.MaaEnd.Path)
	if err := maaWrapper.Init(); err != nil {
		log.Fatalf("初始化 MaaFramework 失败: %v", err)
	}
	defer maaWrapper.Cleanup()

	// 创建 WebSocket 客户端
	wsClient := client.NewClient(cfg)
	wsClient.SetMaaWrapper(&MaaWrapperAdapter{wrapper: maaWrapper})

	// 设置回调
	wsClient.SetCallbacks(
		func() {
			log.Println("[Main] 已连接到服务器")
		},
		func() {
			log.Println("[Main] 与服务器断开连接")
		},
		func(msg *client.Message) {
			// 处理注册成功
			if msg.Type == client.MsgTypeRegistered {
				var payload client.RegisteredPayload
				if err := msg.ParsePayload(&payload); err == nil {
					// 保存到本地存储
					localStorage.SaveCredentials(payload.DeviceID, payload.DeviceToken, cfg.Device.Name)
				}
			}
			// 处理认证失败
			if msg.Type == client.MsgTypeAuthFailed {
				localStorage.ClearCredentials()
			}
		},
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\n收到退出信号，正在关闭...")
		cancel()
		wsClient.Stop()
	}()

	// 处理绑定码
	if *bindCode != "" {
		// 使用命令行参数绑定
		go func() {
			// 等待连接成功
			for !wsClient.IsConnected() {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			wsClient.SendRegister(*bindCode)
		}()
	} else if !wsClient.HasToken() {
		// 没有 token，需要绑定
		fmt.Println("\n设备未绑定，请按以下步骤操作：")
		fmt.Println("1. 在 Web 端获取绑定码")
		fmt.Println("2. 输入绑定码后按回车")
		fmt.Print("\n请输入绑定码: ")

		go func() {
			reader := bufio.NewReader(os.Stdin)
			for {
				code, err := reader.ReadString('\n')
				if err != nil {
					continue
				}
				code = strings.TrimSpace(code)
				if code != "" && wsClient.IsConnected() {
					wsClient.SendRegister(code)
					break
				}
			}
		}()
	}

	// 运行客户端
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

// GetCapabilities 获取设备能力
func (a *MaaWrapperAdapter) GetCapabilities() (*client.CapabilitiesPayload, error) {
	return a.wrapper.GetCapabilities()
}

// RunTask 执行任务
func (a *MaaWrapperAdapter) RunTask(job *client.Job, statusCh chan<- client.TaskStatusPayload, logCh chan<- client.TaskLogPayload) error {
	return a.wrapper.RunTask(job, statusCh, logCh)
}

// StopTask 停止任务
func (a *MaaWrapperAdapter) StopTask() error {
	return a.wrapper.StopTask()
}

// TakeScreenshot 截图
func (a *MaaWrapperAdapter) TakeScreenshot() ([]byte, int, int, error) {
	return a.wrapper.TakeScreenshot()
}

// ClearEventChannels 清除事件通道引用
func (a *MaaWrapperAdapter) ClearEventChannels() {
	a.wrapper.ClearEventChannels()
}
