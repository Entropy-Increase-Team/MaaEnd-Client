package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Version string        `mapstructure:"version"` // 客户端版本号
	Server  ServerConfig  `mapstructure:"server"`
	MaaEnd  MaaEndConfig  `mapstructure:"maaend"`
	Device  DeviceConfig  `mapstructure:"device"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	WsURL             string        `mapstructure:"ws_url"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	ReconnectMaxDelay time.Duration `mapstructure:"reconnect_max_delay"`
}

// MaaEndConfig MaaEnd 配置
type MaaEndConfig struct {
	Path             string `mapstructure:"path"`
	Win32ClassRegex  string `mapstructure:"win32_class_regex"`
	Win32WindowRegex string `mapstructure:"win32_window_regex"`
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	Name  string `mapstructure:"name"`
	Token string `mapstructure:"token"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

var globalConfig *Config
var configFilePath string

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	v.SetDefault("version", "0.2.0")
	v.SetDefault("server.ws_url", "ws://localhost:15618/ws/maaend")
	v.SetDefault("server.connect_timeout", "10s")
	v.SetDefault("server.heartbeat_interval", "30s")
	v.SetDefault("server.reconnect_max_delay", "30s")
	v.SetDefault("maaend.path", "")
	v.SetDefault("maaend.win32_class_regex", "")
	v.SetDefault("maaend.win32_window_regex", "")
	v.SetDefault("device.name", "")
	v.SetDefault("device.token", "")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.file", "")

	configFilePath = resolveConfigPath(configPath)
	v.SetConfigFile(configFilePath)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 确保版本号有值
	if cfg.Version == "" {
		cfg.Version = "0.2.0"
	}

	// 自动检测 MaaEnd 路径
	if cfg.MaaEnd.Path == "" {
		cfg.MaaEnd.Path = detectMaaEndPath()
	}

	// 设置默认设备名
	if cfg.Device.Name == "" {
		hostname, _ := os.Hostname()
		if hostname != "" {
			cfg.Device.Name = hostname
		} else {
			cfg.Device.Name = "MaaEnd-Client"
		}
	}

	globalConfig = &cfg
	return &cfg, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// SaveToken 保存设备令牌到配置文件
func SaveToken(token string) error {
	if globalConfig == nil {
		return fmt.Errorf("配置未加载")
	}

	globalConfig.Device.Token = token

	// 使用模板保存配置，保持格式整洁
	return SaveConfig()
}

// SaveConfig 保存完整配置到文件
func SaveConfig() error {
	if globalConfig == nil {
		return fmt.Errorf("配置未加载")
	}

	path := getConfigFilePath()
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	configContent := fmt.Sprintf(`# MaaEnd Client 配置文件

# 客户端版本号
version: "%s"

server:
  # 云端 WebSocket 地址
  ws_url: "%s"
  # 连接超时
  connect_timeout: %s
  # 心跳间隔
  heartbeat_interval: %s
  # 重连最大延迟
  reconnect_max_delay: %s

maaend:
  # MaaEnd 安装目录（为空则自动检测）
  path: "%s"
  # 覆盖 Win32 窗口类名（正则，留空不覆盖）
  win32_class_regex: "%s"
  # 覆盖 Win32 窗口标题（正则，留空不覆盖）
  win32_window_regex: "%s"

device:
  # 设备名称（为空则使用主机名）
  name: "%s"
  # 设备令牌（首次绑定后自动保存）
  token: "%s"

logging:
  # 日志级别: debug, info, warn, error
  level: "%s"
  # 日志文件（为空则输出到控制台）
  file: "%s"
`,
		globalConfig.Version,
		globalConfig.Server.WsURL,
		globalConfig.Server.ConnectTimeout,
		globalConfig.Server.HeartbeatInterval,
		globalConfig.Server.ReconnectMaxDelay,
		globalConfig.MaaEnd.Path,
		globalConfig.MaaEnd.Win32ClassRegex,
		globalConfig.MaaEnd.Win32WindowRegex,
		globalConfig.Device.Name,
		globalConfig.Device.Token,
		globalConfig.Logging.Level,
		globalConfig.Logging.File,
	)

	return os.WriteFile(path, []byte(configContent), 0644)
}

// EnsureConfigFormat 检查并修复配置文件格式
func EnsureConfigFormat() error {
	// 检查配置文件是否存在
	configFile := getConfigFilePath()
	content, err := os.ReadFile(configFile)
	if err != nil {
		// 文件不存在，创建新的
		return SaveConfig()
	}

	// 检查文件是否以注释开头（正确格式）
	if !strings.HasPrefix(string(content), "# MaaEnd Client") {
		// 格式被破坏，重新保存
		return SaveConfig()
	}

	return nil
}

func resolveConfigPath(configPath string) string {
	if configPath != "" {
		if abs, err := filepath.Abs(configPath); err == nil {
			return abs
		}
		return configPath
	}

	defaultPath := defaultConfigPath()
	if abs, err := filepath.Abs(defaultPath); err == nil {
		return abs
	}
	return defaultPath
}

func defaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(filepath.Dir(exe), "config.yaml")
}

func getConfigFilePath() string {
	if configFilePath != "" {
		return configFilePath
	}
	return defaultConfigPath()
}

// detectMaaEndPath 自动检测 MaaEnd 安装路径
func detectMaaEndPath() string {
	// 1. 检查当前目录
	if isMaaEndDir(".") {
		absPath, _ := filepath.Abs(".")
		return absPath
	}

	// 2. 检查当前目录的父目录（如果 client 在 MaaEnd 目录内）
	if isMaaEndDir("..") {
		absPath, _ := filepath.Abs("..")
		return absPath
	}

	// 3. 检查 %APPDATA%/MaaEnd (Windows)
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			maaEndPath := filepath.Join(appData, "MaaEnd")
			if isMaaEndDir(maaEndPath) {
				return maaEndPath
			}
		}
	}

	// 4. 检查常见位置
	commonPaths := []string{
		"C:/MaaEnd",
		"D:/MaaEnd",
		"E:/MaaEnd",
		filepath.Join(os.Getenv("HOME"), "MaaEnd"),
	}

	for _, p := range commonPaths {
		if isMaaEndDir(p) {
			return p
		}
	}

	return ""
}

// isMaaEndDir 检查目录是否是 MaaEnd 安装目录
func isMaaEndDir(path string) bool {
	// 检查 interface.json 是否存在
	interfacePath := filepath.Join(path, "interface.json")
	if _, err := os.Stat(interfacePath); err != nil {
		return false
	}

	// 检查 maafw 目录是否存在
	maafwPath := filepath.Join(path, "maafw")
	if info, err := os.Stat(maafwPath); err != nil || !info.IsDir() {
		return false
	}

	return true
}

// GetOSInfo 获取操作系统信息
func GetOSInfo() string {
	var osInfo strings.Builder
	osInfo.WriteString(runtime.GOOS)
	osInfo.WriteString(" ")
	osInfo.WriteString(runtime.GOARCH)

	// Windows 版本信息
	if runtime.GOOS == "windows" {
		// 简单的版本检测
		osInfo.WriteString(" (")
		if _, err := os.Stat("C:\\Windows\\System32\\win32kbase.sys"); err == nil {
			osInfo.WriteString("Windows 10+")
		} else {
			osInfo.WriteString("Windows")
		}
		osInfo.WriteString(")")
	}

	return osInfo.String()
}
