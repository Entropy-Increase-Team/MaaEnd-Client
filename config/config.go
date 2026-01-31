package config

import (
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

// Load 加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
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

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认配置文件位置
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
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

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// 读取现有配置
	v.ReadInConfig()

	// 更新 token
	v.Set("device.token", token)

	// 写入文件
	return v.WriteConfig()
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
