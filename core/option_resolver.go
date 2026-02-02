package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// OptionResolver 选项解析器
type OptionResolver struct {
	pi *ProjectInterface
}

// NewOptionResolver 创建选项解析器
func NewOptionResolver(pi *ProjectInterface) *OptionResolver {
	return &OptionResolver{pi: pi}
}

// ResolveTaskOptions 解析任务选项，构建 pipeline_override
func (r *OptionResolver) ResolveTaskOptions(taskName string, userOptions map[string]interface{}) (map[string]interface{}, error) {
	task := r.pi.GetTask(taskName)
	if task == nil {
		return nil, fmt.Errorf("任务不存在: %s", taskName)
	}

	// 合并 override
	override := make(map[string]interface{})

	// 首先应用任务级别的 pipeline_override
	if task.PipelineOverride != nil {
		mergeOverride(override, task.PipelineOverride)
	}

	// 处理每个 option
	for _, optName := range task.Option {
		opt := r.pi.GetOption(optName)
		if opt == nil {
			continue
		}

		// 获取用户选择
		userValue := userOptions[optName]

		// 解析选项
		optOverride, err := r.resolveOption(optName, opt, userValue, userOptions)
		if err != nil {
			return nil, fmt.Errorf("解析选项 %s 失败: %w", optName, err)
		}

		if optOverride != nil {
			mergeOverride(override, optOverride)
		}
	}

	return override, nil
}

// resolveOption 解析单个选项
func (r *OptionResolver) resolveOption(name string, opt *OptionConfig, userValue interface{}, allUserOptions map[string]interface{}) (map[string]interface{}, error) {
	switch opt.Type {
	case "select":
		return r.resolveSelectOption(name, opt, userValue, allUserOptions)
	case "switch":
		// switch 类型与 select 逻辑相同，只是通常只有 Yes/No 两个选项
		return r.resolveSelectOption(name, opt, userValue, allUserOptions)
	case "checkbox":
		return r.resolveCheckboxOption(name, opt, userValue, allUserOptions)
	case "input":
		return r.resolveInputOption(name, opt, userValue, allUserOptions)
	default:
		return nil, fmt.Errorf("未知选项类型: %s", opt.Type)
	}
}

// resolveSelectOption 解析选择类型选项（也用于 switch 类型）
func (r *OptionResolver) resolveSelectOption(name string, opt *OptionConfig, userValue interface{}, allUserOptions map[string]interface{}) (map[string]interface{}, error) {
	override := make(map[string]interface{})

	// 获取选中的 case
	// 优先级：用户选择 > DefaultCase > Default > 第一个 case
	selectedCase := opt.DefaultCase
	if selectedCase == "" && opt.Default != "" {
		selectedCase = opt.Default
	}
	if userValue != nil {
		if strVal, ok := userValue.(string); ok {
			selectedCase = strVal
		}
	}

	// 如果仍然为空，使用第一个 case 作为默认
	if selectedCase == "" && len(opt.Cases) > 0 {
		selectedCase = opt.Cases[0].Name
	}

	// 查找对应的 case
	var caseConfig *CaseConfig
	for i := range opt.Cases {
		if opt.Cases[i].Name == selectedCase {
			caseConfig = &opt.Cases[i]
			break
		}
	}

	if caseConfig == nil {
		// 如果没有找到且有 cases，返回空 override 而非错误
		if len(opt.Cases) == 0 {
			return override, nil
		}
		return nil, fmt.Errorf("选项 %s 的 case %s 不存在", name, selectedCase)
	}

	// 应用 case 的 pipeline_override
	if caseConfig.PipelineOverride != nil {
		mergeOverride(override, caseConfig.PipelineOverride)
	}

	// 递归处理嵌套选项
	for _, nestedOptName := range caseConfig.Option {
		nestedOpt := r.pi.GetOption(nestedOptName)
		if nestedOpt == nil {
			continue
		}

		nestedValue := allUserOptions[nestedOptName]
		nestedOverride, err := r.resolveOption(nestedOptName, nestedOpt, nestedValue, allUserOptions)
		if err != nil {
			return nil, fmt.Errorf("解析嵌套选项 %s 失败: %w", nestedOptName, err)
		}

		if nestedOverride != nil {
			mergeOverride(override, nestedOverride)
		}
	}

	return override, nil
}

// resolveCheckboxOption 解析复选框类型选项
func (r *OptionResolver) resolveCheckboxOption(_ string, opt *OptionConfig, userValue interface{}, allUserOptions map[string]interface{}) (map[string]interface{}, error) {
	override := make(map[string]interface{})

	// 获取选中的 cases
	var selectedCases []string
	if userValue != nil {
		switch v := userValue.(type) {
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					selectedCases = append(selectedCases, str)
				}
			}
		case []string:
			selectedCases = v
		}
	}

	// 如果没有选择，使用默认值
	if len(selectedCases) == 0 && opt.DefaultCase != "" {
		selectedCases = strings.Split(opt.DefaultCase, ",")
	}

	// 处理每个选中的 case
	for _, caseName := range selectedCases {
		caseName = strings.TrimSpace(caseName)
		var caseConfig *CaseConfig
		for i := range opt.Cases {
			if opt.Cases[i].Name == caseName {
				caseConfig = &opt.Cases[i]
				break
			}
		}

		if caseConfig == nil {
			continue
		}

		// 应用 case 的 pipeline_override
		if caseConfig.PipelineOverride != nil {
			mergeOverride(override, caseConfig.PipelineOverride)
		}

		// 递归处理嵌套选项
		for _, nestedOptName := range caseConfig.Option {
			nestedOpt := r.pi.GetOption(nestedOptName)
			if nestedOpt == nil {
				continue
			}

			nestedValue := allUserOptions[nestedOptName]
			nestedOverride, err := r.resolveOption(nestedOptName, nestedOpt, nestedValue, allUserOptions)
			if err != nil {
				return nil, fmt.Errorf("解析嵌套选项 %s 失败: %w", nestedOptName, err)
			}

			if nestedOverride != nil {
				mergeOverride(override, nestedOverride)
			}
		}
	}

	return override, nil
}

// resolveInputOption 解析输入类型选项
func (r *OptionResolver) resolveInputOption(_ string, opt *OptionConfig, userValue interface{}, _ map[string]interface{}) (map[string]interface{}, error) {
	override := make(map[string]interface{})

	// 获取输入值
	inputValues := make(map[string]string)

	// 从 inputs 配置获取默认值
	for _, input := range opt.Inputs {
		defaultVal := input.GetDefaultString()
		if defaultVal != "" {
			inputValues[input.Name] = defaultVal
		}
	}

	// 覆盖用户输入
	if userValue != nil {
		switch v := userValue.(type) {
		case map[string]interface{}:
			for k, val := range v {
				inputValues[k] = formatInputValue(val)
			}
		case map[string]string:
			for k, val := range v {
				inputValues[k] = val
			}
		}
	}

	// 应用 pipeline_override 并进行变量替换
	if opt.PipelineOverride != nil {
		resolved := resolveVariables(opt.PipelineOverride, inputValues)
		mergeOverride(override, resolved)
	}

	return override, nil
}

// resolveVariables 替换 pipeline_override 中的变量
func resolveVariables(override map[string]interface{}, values map[string]string) map[string]interface{} {
	// 深拷贝
	data, _ := json.Marshal(override)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	// 递归替换
	resolveVariablesRecursive(result, values)

	return result
}

// resolveVariablesRecursive 递归替换变量
func resolveVariablesRecursive(data interface{}, values map[string]string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			switch vv := val.(type) {
			case string:
				v[key] = replaceVariables(vv, values)
			case map[string]interface{}:
				resolveVariablesRecursive(vv, values)
			case []interface{}:
				resolveVariablesRecursive(vv, values)
			}
		}
	case []interface{}:
		for i, item := range v {
			switch vv := item.(type) {
			case string:
				v[i] = replaceVariables(vv, values)
			case map[string]interface{}:
				resolveVariablesRecursive(vv, values)
			case []interface{}:
				resolveVariablesRecursive(vv, values)
			}
		}
	}
}

// replaceVariables 替换字符串中的变量
func replaceVariables(s string, values map[string]string) string {
	// 匹配 {varName} 格式
	re := regexp.MustCompile(`\{(\w+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[1 : len(match)-1]
		if val, ok := values[varName]; ok {
			return val
		}
		return match
	})
}

// mergeOverride 合并 pipeline_override
func mergeOverride(dst, src map[string]interface{}) {
	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// 如果都是 map，递归合并
			if dstMap, ok := dstVal.(map[string]interface{}); ok {
				if srcMap, ok := srcVal.(map[string]interface{}); ok {
					mergeOverride(dstMap, srcMap)
					continue
				}
			}
		}
		// 否则直接覆盖
		dst[key] = srcVal
	}
}

// formatInputValue 格式化输入值为字符串
func formatInputValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON 数字解析为 float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
