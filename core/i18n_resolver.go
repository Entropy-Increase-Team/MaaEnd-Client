package core

import (
	"encoding/json"
	"fmt"
)

// ResolveControllerForEnv 把指定控制器配置中所有 i18n（以 $ 开头的字段）解析为展示文本，
// 返回紧凑 JSON，用于 PI v2.5.0 的 PI_CONTROLLER 环境变量。
// 若找不到控制器则返回空字符串与错误。
func ResolveControllerForEnv(pi *ProjectInterface, controllerName, lang string) (string, error) {
	if pi == nil {
		return "", fmt.Errorf("project interface 未加载")
	}
	ctrl := pi.GetController(controllerName)
	if ctrl == nil {
		return "", fmt.Errorf("控制器不存在: %s", controllerName)
	}

	// 通过 json 序列化 + 反序列化得到通用 map，再对 i18n 字段做就地替换
	raw, err := json.Marshal(ctrl)
	if err != nil {
		return "", fmt.Errorf("序列化控制器失败: %w", err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("反序列化控制器失败: %w", err)
	}

	resolveI18nStringsInPlace(obj, pi, lang)

	resolved, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("重新序列化控制器失败: %w", err)
	}
	return string(resolved), nil
}

// ResolveResourceForEnv 与 ResolveControllerForEnv 类似，用于 PI_RESOURCE 环境变量。
func ResolveResourceForEnv(pi *ProjectInterface, resourceName, lang string) (string, error) {
	if pi == nil {
		return "", fmt.Errorf("project interface 未加载")
	}
	res := pi.GetResource(resourceName)
	if res == nil {
		return "", fmt.Errorf("资源不存在: %s", resourceName)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化资源失败: %w", err)
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", fmt.Errorf("反序列化资源失败: %w", err)
	}

	resolveI18nStringsInPlace(obj, pi, lang)

	resolved, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("重新序列化资源失败: %w", err)
	}
	return string(resolved), nil
}

// resolveI18nStringsInPlace 递归遍历并把所有以 `$` 开头的字符串替换为 i18n 解析结果。
// 这是一个通用 helper，不依赖特定字段名，保证后续新增 i18n 字段时不需要同步修改。
func resolveI18nStringsInPlace(v interface{}, pi *ProjectInterface, lang string) {
	switch node := v.(type) {
	case map[string]interface{}:
		for k, val := range node {
			switch inner := val.(type) {
			case string:
				node[k] = pi.GetI18nString(inner, lang)
			case map[string]interface{}, []interface{}:
				resolveI18nStringsInPlace(inner, pi, lang)
			}
		}
	case []interface{}:
		for i, val := range node {
			switch inner := val.(type) {
			case string:
				node[i] = pi.GetI18nString(inner, lang)
			case map[string]interface{}, []interface{}:
				resolveI18nStringsInPlace(inner, pi, lang)
			}
		}
	}
}
