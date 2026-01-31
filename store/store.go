package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store 本地存储
type Store struct {
	path string
	data *StoreData
	mu   sync.RWMutex
}

// StoreData 存储数据
type StoreData struct {
	DeviceID    string            `json:"device_id"`
	DeviceToken string            `json:"device_token"`
	DeviceName  string            `json:"device_name"`
	Extra       map[string]string `json:"extra,omitempty"`
}

var (
	globalStore *Store
	once        sync.Once
)

// GetStore 获取全局存储实例
func GetStore() *Store {
	once.Do(func() {
		globalStore = NewStore("")
	})
	return globalStore
}

// NewStore 创建存储实例
func NewStore(path string) *Store {
	if path == "" {
		// 默认存储路径
		exe, _ := os.Executable()
		path = filepath.Join(filepath.Dir(exe), "device.json")
	}

	s := &Store{
		path: path,
		data: &StoreData{
			Extra: make(map[string]string),
		},
	}

	// 加载已有数据
	s.load()

	return s
}

// load 加载数据
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, s.data)
}

// save 保存数据
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// GetDeviceID 获取设备ID
func (s *Store) GetDeviceID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.DeviceID
}

// SetDeviceID 设置设备ID
func (s *Store) SetDeviceID(id string) error {
	s.mu.Lock()
	s.data.DeviceID = id
	s.mu.Unlock()
	return s.save()
}

// GetDeviceToken 获取设备令牌
func (s *Store) GetDeviceToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.DeviceToken
}

// SetDeviceToken 设置设备令牌
func (s *Store) SetDeviceToken(token string) error {
	s.mu.Lock()
	s.data.DeviceToken = token
	s.mu.Unlock()
	return s.save()
}

// GetDeviceName 获取设备名称
func (s *Store) GetDeviceName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.DeviceName
}

// SetDeviceName 设置设备名称
func (s *Store) SetDeviceName(name string) error {
	s.mu.Lock()
	s.data.DeviceName = name
	s.mu.Unlock()
	return s.save()
}

// GetExtra 获取额外数据
func (s *Store) GetExtra(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Extra[key]
}

// SetExtra 设置额外数据
func (s *Store) SetExtra(key, value string) error {
	s.mu.Lock()
	if s.data.Extra == nil {
		s.data.Extra = make(map[string]string)
	}
	s.data.Extra[key] = value
	s.mu.Unlock()
	return s.save()
}

// SaveCredentials 保存凭证
func (s *Store) SaveCredentials(deviceID, deviceToken, deviceName string) error {
	s.mu.Lock()
	s.data.DeviceID = deviceID
	s.data.DeviceToken = deviceToken
	s.data.DeviceName = deviceName
	s.mu.Unlock()
	return s.save()
}

// ClearCredentials 清除凭证
func (s *Store) ClearCredentials() error {
	s.mu.Lock()
	s.data.DeviceID = ""
	s.data.DeviceToken = ""
	s.mu.Unlock()
	return s.save()
}

// HasCredentials 检查是否有凭证
func (s *Store) HasCredentials() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.DeviceToken != ""
}
