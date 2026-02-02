package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectInterface interface.json 的完整结构
type ProjectInterface struct {
	InterfaceVersion int                      `json:"interface_version"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Version          string                   `json:"version"`
	Contact          string                   `json:"contact"`
	License          string                   `json:"license"`
	Welcome          string                   `json:"welcome"`
	Icon             string                   `json:"icon"`
	GitHub           string                   `json:"github"`
	Languages        map[string]string        `json:"languages"`
	Controllers      []ControllerConfig       `json:"controller"`
	Resources        []ResourceConfig         `json:"resource"`
	Agent            AgentConfig              `json:"agent"`
	Tasks            []TaskConfig             `json:"task"`
	Options          map[string]*OptionConfig `json:"option"`
	Import           []string                 `json:"import"` // 外部任务文件路径列表

	// 解析后的国际化文本
	i18nTexts map[string]map[string]string // lang -> key -> value
	basePath  string                       // interface.json 所在目录
}

// ControllerConfig 控制器配置
type ControllerConfig struct {
	Name               string       `json:"name"`
	Label              string       `json:"label"`
	Type               string       `json:"type"`
	Win32              *Win32Config `json:"win32,omitempty"`
	PermissionRequired bool         `json:"permission_required"`
}

// Win32Config Win32 控制器配置
type Win32Config struct {
	ClassRegex  string `json:"class_regex"`
	WindowRegex string `json:"window_regex"`
	Screencap   string `json:"screencap"`
	Mouse       string `json:"mouse"`
	Keyboard    string `json:"keyboard"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	Name string   `json:"name"`
	Path []string `json:"path"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	ChildExec string   `json:"child_exec"`
	ChildArgs []string `json:"child_args"`
}

// TaskConfig 任务配置
type TaskConfig struct {
	Name             string                 `json:"name"`
	Label            string                 `json:"label"`
	Entry            string                 `json:"entry"`
	Description      string                 `json:"description"`
	DefaultCheck     bool                   `json:"default_check,omitempty"` // 默认选中
	Option           []string               `json:"option"`
	Controller       []string               `json:"controller"`
	Resource         []string               `json:"resource"`
	PipelineOverride map[string]interface{} `json:"pipeline_override"`
}

// OptionConfig 选项配置
type OptionConfig struct {
	Type             string                 `json:"type"`
	Label            string                 `json:"label"`
	Description      string                 `json:"description"`
	Default          string                 `json:"default,omitempty"`
	Cases            []CaseConfig           `json:"cases,omitempty"`
	Inputs           []InputConfig          `json:"inputs,omitempty"`
	DefaultCase      string                 `json:"default_case,omitempty"`
	PipelineOverride map[string]interface{} `json:"pipeline_override,omitempty"`
}

// CaseConfig 选项分支配置
type CaseConfig struct {
	Name             string                 `json:"name"`
	Label            string                 `json:"label"`
	Option           []string               `json:"option,omitempty"`
	PipelineOverride map[string]interface{} `json:"pipeline_override,omitempty"`
}

// InputConfig 输入配置
type InputConfig struct {
	Name         string      `json:"name"`
	Label        string      `json:"label"`
	Description  string      `json:"description,omitempty"`
	PipelineType string      `json:"pipeline_type,omitempty"`
	Verify       string      `json:"verify,omitempty"`
	Default      interface{} `json:"default,omitempty"`
}

// GetDefaultString 获取默认值的字符串形式
func (i *InputConfig) GetDefaultString() string {
	if i.Default == nil {
		return ""
	}
	switch v := i.Default.(type) {
	case string:
		return v
	case float64:
		// JSON 数字解析为 float64
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// LoadInterface 加载 interface.json
func LoadInterface(maaEndPath string) (*ProjectInterface, error) {
	interfacePath := filepath.Join(maaEndPath, "interface.json")

	data, err := os.ReadFile(interfacePath)
	if err != nil {
		return nil, fmt.Errorf("读取 interface.json 失败: %w", err)
	}

	var pi ProjectInterface
	if err := json.Unmarshal(data, &pi); err != nil {
		return nil, fmt.Errorf("解析 interface.json 失败: %w", err)
	}

	pi.basePath = maaEndPath
	pi.i18nTexts = make(map[string]map[string]string)

	// 初始化 Options map（如果为空）
	if pi.Options == nil {
		pi.Options = make(map[string]*OptionConfig)
	}

	// 加载 import 引用的外部任务文件
	for _, importPath := range pi.Import {
		if err := pi.loadImportedFile(importPath); err != nil {
			// import 加载失败记录警告但不中断
			fmt.Printf("警告: 加载导入文件 %s 失败: %v\n", importPath, err)
		}
	}

	// 加载国际化文件
	for lang, path := range pi.Languages {
		if err := pi.loadI18n(lang, path); err != nil {
			// 国际化加载失败不是致命错误
			fmt.Printf("警告: 加载国际化文件 %s 失败: %v\n", path, err)
		}
	}

	return &pi, nil
}

// ImportedFile 外部导入文件的结构
type ImportedFile struct {
	Tasks   []TaskConfig             `json:"task"`
	Options map[string]*OptionConfig `json:"option"`
}

// loadImportedFile 加载单个导入文件，合并 task 和 option
func (pi *ProjectInterface) loadImportedFile(relativePath string) error {
	fullPath := filepath.Join(pi.basePath, relativePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var imported ImportedFile
	if err := json.Unmarshal(data, &imported); err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	// 合并任务
	pi.Tasks = append(pi.Tasks, imported.Tasks...)

	// 合并选项
	for name, opt := range imported.Options {
		if opt != nil {
			pi.Options[name] = opt
		}
	}

	return nil
}

// loadI18n 加载国际化文件
func (pi *ProjectInterface) loadI18n(lang, path string) error {
	fullPath := filepath.Join(pi.basePath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	var texts map[string]string
	if err := json.Unmarshal(data, &texts); err != nil {
		return err
	}

	pi.i18nTexts[lang] = texts
	return nil
}

// GetI18nString 获取国际化字符串
func (pi *ProjectInterface) GetI18nString(key, lang string) string {
	// 如果不是国际化 key（不以 $ 开头），直接返回
	if !strings.HasPrefix(key, "$") {
		return key
	}

	// 去掉 $ 前缀
	realKey := key[1:]

	// 尝试获取指定语言
	if texts, ok := pi.i18nTexts[lang]; ok {
		if text, ok := texts[realKey]; ok {
			return text
		}
	}

	// 回退到中文
	if texts, ok := pi.i18nTexts["zh_cn"]; ok {
		if text, ok := texts[realKey]; ok {
			return text
		}
	}

	// 回退到第一个可用语言
	for _, texts := range pi.i18nTexts {
		if text, ok := texts[realKey]; ok {
			return text
		}
	}

	// 返回原始 key
	return key
}

// GetControllerNames 获取所有控制器名称
func (pi *ProjectInterface) GetControllerNames() []string {
	names := make([]string, len(pi.Controllers))
	for i, c := range pi.Controllers {
		names[i] = c.Name
	}
	return names
}

// GetResourceNames 获取所有资源名称
func (pi *ProjectInterface) GetResourceNames() []string {
	names := make([]string, len(pi.Resources))
	for i, r := range pi.Resources {
		names[i] = r.Name
	}
	return names
}

// GetController 根据名称获取控制器配置
func (pi *ProjectInterface) GetController(name string) *ControllerConfig {
	for i := range pi.Controllers {
		if pi.Controllers[i].Name == name {
			return &pi.Controllers[i]
		}
	}
	return nil
}

// GetResource 根据名称获取资源配置
func (pi *ProjectInterface) GetResource(name string) *ResourceConfig {
	for i := range pi.Resources {
		if pi.Resources[i].Name == name {
			return &pi.Resources[i]
		}
	}
	return nil
}

// GetTask 根据名称获取任务配置
func (pi *ProjectInterface) GetTask(name string) *TaskConfig {
	for i := range pi.Tasks {
		if pi.Tasks[i].Name == name {
			return &pi.Tasks[i]
		}
	}
	return nil
}

// GetOption 根据名称获取选项配置
func (pi *ProjectInterface) GetOption(name string) *OptionConfig {
	return pi.Options[name]
}

// GetBasePath 获取基础路径
func (pi *ProjectInterface) GetBasePath() string {
	return pi.basePath
}

// GetAgentExec 获取 Agent 可执行文件完整路径
func (pi *ProjectInterface) GetAgentExec() string {
	if pi.Agent.ChildExec == "" {
		return ""
	}
	return filepath.Join(pi.basePath, pi.Agent.ChildExec)
}

// GetMaaFWPath 获取 MaaFramework 库路径
func (pi *ProjectInterface) GetMaaFWPath() string {
	return filepath.Join(pi.basePath, "maafw")
}

// GetResourcePaths 获取资源的完整路径列表
func (pi *ProjectInterface) GetResourcePaths(name string) []string {
	res := pi.GetResource(name)
	if res == nil {
		return nil
	}

	paths := make([]string, len(res.Path))
	for i, p := range res.Path {
		if filepath.IsAbs(p) {
			paths[i] = p
		} else {
			paths[i] = filepath.Join(pi.basePath, p)
		}
	}
	return paths
}
