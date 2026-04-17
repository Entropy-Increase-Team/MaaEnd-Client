package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// stripJSONComments removes single-line (//) and multi-line (/* */) comments
// from JSON data, respecting string literals so that "//" inside strings is preserved.
func stripJSONComments(data []byte) []byte {
	var result []byte
	i := 0
	n := len(data)

	for i < n {
		if data[i] == '"' {
			// Inside a string literal — copy until the closing unescaped quote
			result = append(result, data[i])
			i++
			for i < n {
				if data[i] == '\\' && i+1 < n {
					result = append(result, data[i], data[i+1])
					i += 2
					continue
				}
				result = append(result, data[i])
				if data[i] == '"' {
					i++
					break
				}
				i++
			}
		} else if i+1 < n && data[i] == '/' && data[i+1] == '/' {
			// Single-line comment — skip until newline
			i += 2
			for i < n && data[i] != '\n' {
				i++
			}
		} else if i+1 < n && data[i] == '/' && data[i+1] == '*' {
			// Multi-line comment — skip until */
			i += 2
			for i+1 < n {
				if data[i] == '*' && data[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		} else {
			result = append(result, data[i])
			i++
		}
	}

	return result
}

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
	AgentRaw         json.RawMessage          `json:"agent"`
	Tasks            []TaskConfig             `json:"task"`
	Options          map[string]*OptionConfig `json:"option"`
	GlobalOption     []string                 `json:"global_option"`
	Import           []string                 `json:"import"` // 外部任务文件路径列表

	Agent   []AgentConfig  `json:"-"` // 统一解析后的 Agent 列表
	Presets []PresetConfig `json:"-"` // 从导入文件中解析的 Preset

	// 解析后的国际化文本
	i18nTexts map[string]map[string]string // lang -> key -> value
	basePath  string                       // interface.json 所在目录
}

// PresetConfig 预设任务组配置
type PresetConfig struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Tasks       []PresetTask `json:"task"`
}

// PresetTask 预设中的单个任务
type PresetTask struct {
	Name    string                 `json:"name"`
	Enabled *bool                  `json:"enabled,omitempty"`
	Option  map[string]interface{} `json:"option,omitempty"`
}

// ControllerConfig 控制器配置
type ControllerConfig struct {
	Name               string           `json:"name"`
	Label              string           `json:"label"`
	Description        string           `json:"description,omitempty"`
	Type               string           `json:"type"`
	Win32              *Win32Config     `json:"win32,omitempty"`
	Adb                *AdbConfig       `json:"adb,omitempty"`
	PlayCover          *PlayCoverConfig `json:"playcover,omitempty"`
	AttachResourcePath []string         `json:"attach_resource_path,omitempty"`
	Option             []string         `json:"option,omitempty"`
	PermissionRequired bool             `json:"permission_required"`

	// PI V2 协议截图分辨率字段（三者互斥；Client 应按 display_raw > long_side > short_side 顺序取值，
	// 都未设置时按协议默认短边 720）
	DisplayShortSide int32 `json:"display_short_side,omitempty"`
	DisplayLongSide  int32 `json:"display_long_side,omitempty"`
	DisplayRaw       bool  `json:"display_raw,omitempty"`
}

// AdbConfig ADB 控制器配置
type AdbConfig struct {
	Address       string `json:"address,omitempty"`
	Config        string `json:"config,omitempty"`
	ScreencapType int    `json:"screencap_type,omitempty"`
	InputType     int    `json:"input_type,omitempty"`
}

// PlayCoverConfig PlayCover 控制器配置
type PlayCoverConfig struct {
	UUID string `json:"uuid,omitempty"`
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
	Name       string   `json:"name"`
	Label      string   `json:"label,omitempty"`
	Path       []string `json:"path"`
	Controller []string `json:"controller,omitempty"`
	Option     []string `json:"option,omitempty"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	ChildExec string   `json:"child_exec"`
	ChildArgs []string `json:"child_args"`
	// Identifier 可选，用于创建通信套接字标识符。留空时由 Client 自动生成。
	// 参考 MaaFW PI V2 "agent.identifier" 字段。
	Identifier string `json:"identifier,omitempty"`
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
	DefaultCase      interface{}            `json:"default_case,omitempty"`
	Controller       []string               `json:"controller,omitempty"`
	Resource         []string               `json:"resource,omitempty"`
	Option           []string               `json:"option,omitempty"`
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
	PatternMsg   string      `json:"pattern_msg,omitempty"`
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

	data = stripJSONComments(data)

	var pi ProjectInterface
	if err := json.Unmarshal(data, &pi); err != nil {
		return nil, fmt.Errorf("解析 interface.json 失败: %w", err)
	}

	if err := normalizeAgentConfigs(&pi); err != nil {
		return nil, fmt.Errorf("解析 agent 配置失败: %w", err)
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

func normalizeAgentConfigs(pi *ProjectInterface) error {
	if pi == nil || len(pi.AgentRaw) == 0 {
		pi.Agent = nil
		return nil
	}

	trimmed := bytes.TrimSpace(pi.AgentRaw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		pi.Agent = nil
		return nil
	}

	switch trimmed[0] {
	case '{':
		var single AgentConfig
		if err := json.Unmarshal(trimmed, &single); err != nil {
			return err
		}
		pi.Agent = []AgentConfig{single}
	case '[':
		var list []AgentConfig
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		pi.Agent = list
	default:
		return fmt.Errorf("agent 字段必须是对象或对象数组")
	}

	return nil
}

func getDefaultCaseList(opt *OptionConfig) []string {
	if opt == nil || opt.DefaultCase == nil {
		return nil
	}

	switch v := opt.DefaultCase.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				result = append(result, s)
			}
		}
		return result
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					result = append(result, s)
				}
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(item); s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func getDefaultCaseFirst(opt *OptionConfig) string {
	cases := getDefaultCaseList(opt)
	if len(cases) == 0 {
		return ""
	}
	return cases[0]
}

// ImportedFile 外部导入文件的结构
type ImportedFile struct {
	Tasks   []TaskConfig             `json:"task"`
	Options map[string]*OptionConfig `json:"option"`
	Presets []PresetConfig           `json:"preset"`
}

// loadImportedFile 加载单个导入文件，合并 task 和 option
func (pi *ProjectInterface) loadImportedFile(relativePath string) error {
	fullPath := filepath.Join(pi.basePath, relativePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	data = stripJSONComments(data)

	var imported ImportedFile
	if err := json.Unmarshal(data, &imported); err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	// 合并任务
	pi.Tasks = mergeTasksByName(pi.Tasks, imported.Tasks)

	// 合并选项
	for name, opt := range imported.Options {
		if opt != nil {
			pi.Options[name] = opt
		}
	}

	// 合并预设
	pi.Presets = mergePresetsByName(pi.Presets, imported.Presets)

	return nil
}

func mergeTasksByName(base []TaskConfig, incoming []TaskConfig) []TaskConfig {
	if len(incoming) == 0 {
		return base
	}

	result := make([]TaskConfig, len(base))
	copy(result, base)
	index := make(map[string]int, len(result))
	for i, t := range result {
		if t.Name != "" {
			index[t.Name] = i
		}
	}

	for _, t := range incoming {
		if t.Name == "" {
			result = append(result, t)
			continue
		}
		if i, ok := index[t.Name]; ok {
			result[i] = t
			continue
		}
		index[t.Name] = len(result)
		result = append(result, t)
	}

	return result
}

func mergePresetsByName(base []PresetConfig, incoming []PresetConfig) []PresetConfig {
	if len(incoming) == 0 {
		return base
	}

	result := make([]PresetConfig, len(base))
	copy(result, base)
	index := make(map[string]int, len(result))
	for i, p := range result {
		if p.Name != "" {
			index[p.Name] = i
		}
	}

	for _, p := range incoming {
		if p.Name == "" {
			result = append(result, p)
			continue
		}
		if i, ok := index[p.Name]; ok {
			result[i] = p
			continue
		}
		index[p.Name] = len(result)
		result = append(result, p)
	}

	return result
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

// GetPreset 根据名称获取预设配置
func (pi *ProjectInterface) GetPreset(name string) *PresetConfig {
	for i := range pi.Presets {
		if pi.Presets[i].Name == name {
			return &pi.Presets[i]
		}
	}
	return nil
}

// GetBasePath 获取基础路径
func (pi *ProjectInterface) GetBasePath() string {
	return pi.basePath
}

// GetAgents 获取所有 Agent 配置（带完整路径）
func (pi *ProjectInterface) GetAgents() []AgentConfig {
	return pi.Agent
}

// GetAgentExec 获取第一个 Agent 的可执行文件完整路径（向后兼容）
func (pi *ProjectInterface) GetAgentExec() string {
	if len(pi.Agent) == 0 || pi.Agent[0].ChildExec == "" {
		return ""
	}
	return filepath.Join(pi.basePath, pi.Agent[0].ChildExec)
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
