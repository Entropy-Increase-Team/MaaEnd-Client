package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveControllerForEnv_I18nFieldsExpanded(t *testing.T) {
	pi := &ProjectInterface{
		i18nTexts: map[string]map[string]string{
			"zh_cn": {
				"controller.Win32-Window.label":       "Win32-默认",
				"controller.Win32-Window.description": "默认控制器",
			},
		},
		Controllers: []ControllerConfig{
			{
				Name:        "Win32-Window",
				Label:       "$controller.Win32-Window.label",
				Description: "$controller.Win32-Window.description",
				Type:        "Win32",
				Win32: &Win32Config{
					ClassRegex:  "UnityWndClass",
					WindowRegex: "Endfield",
					Screencap:   "Background",
					Mouse:       "SendMessageWithCursorPos",
					Keyboard:    "PostMessage",
				},
				PermissionRequired: true,
			},
		},
	}

	encoded, err := ResolveControllerForEnv(pi, "Win32-Window", "zh_cn")
	if err != nil {
		t.Fatalf("ResolveControllerForEnv failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("returned value should be valid JSON: %v", err)
	}

	if got := decoded["label"]; got != "Win32-默认" {
		t.Fatalf("label should be i18n-resolved, got %v", got)
	}
	if got := decoded["description"]; got != "默认控制器" {
		t.Fatalf("description should be i18n-resolved, got %v", got)
	}
	if got := decoded["type"]; got != "Win32" {
		t.Fatalf("type should be preserved, got %v", got)
	}

	// 确认 JSON 为紧凑单行（符合 PI v2.5.0 约束）
	if strings.ContainsAny(encoded, "\n\r") {
		t.Fatalf("env json must be single-line compact, got: %q", encoded)
	}

	win32, ok := decoded["win32"].(map[string]interface{})
	if !ok {
		t.Fatalf("nested win32 object should be preserved, got %#v", decoded["win32"])
	}
	if got := win32["screencap"]; got != "Background" {
		t.Fatalf("nested non-i18n field should be preserved, got %v", got)
	}
}

func TestResolveResourceForEnv_FallbackToChineseWhenLangMissing(t *testing.T) {
	pi := &ProjectInterface{
		i18nTexts: map[string]map[string]string{
			"zh_cn": {"resource.CN.label": "官服"},
		},
		Resources: []ResourceConfig{
			{
				Name:       "CN",
				Label:      "$resource.CN.label",
				Path:       []string{"./resource"},
				Controller: []string{"Win32-Window"},
			},
		},
	}

	encoded, err := ResolveResourceForEnv(pi, "CN", "ja_jp")
	if err != nil {
		t.Fatalf("ResolveResourceForEnv failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("returned value should be valid JSON: %v", err)
	}

	if got := decoded["label"]; got != "官服" {
		t.Fatalf("missing lang should fall back to zh_cn, got %v", got)
	}
}

func TestResolveControllerForEnv_UnknownController(t *testing.T) {
	pi := &ProjectInterface{Controllers: nil}
	if _, err := ResolveControllerForEnv(pi, "NotExist", "zh_cn"); err == nil {
		t.Fatalf("expected error for unknown controller")
	}
}
