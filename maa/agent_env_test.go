package maa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"maaend-client/core"
)

// newTestProjectInterface 写出一份最小可用的 interface.json + zh_cn 翻译文件到临时目录
// 并返回加载后的 ProjectInterface，用于 env 构造相关单测。
func newTestProjectInterface(t *testing.T) *core.ProjectInterface {
	t.Helper()

	dir := t.TempDir()

	interfaceJSON := `{
    "interface_version": 2,
    "name": "demo",
    "version": "v1.0.0",
    "languages": {"zh_cn": "locales/zh_cn.json"},
    "controller": [
        {
            "name": "Win32-Window",
            "label": "$ctrl.label",
            "description": "$ctrl.desc",
            "type": "Win32",
            "win32": {
                "class_regex": "UnityWndClass",
                "window_regex": "Endfield",
                "screencap": "Background"
            },
            "permission_required": true
        }
    ],
    "resource": [
        {"name": "CN", "label": "$res.label", "path": ["./resource"]}
    ],
    "task": []
}`
	localeJSON := `{
    "ctrl.label": "Win32-默认",
    "ctrl.desc": "默认控制器",
    "res.label": "官服"
}`

	if err := os.WriteFile(filepath.Join(dir, "interface.json"), []byte(interfaceJSON), 0644); err != nil {
		t.Fatalf("write interface.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "locales"), 0755); err != nil {
		t.Fatalf("mkdir locales: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "locales", "zh_cn.json"), []byte(localeJSON), 0644); err != nil {
		t.Fatalf("write locale: %v", err)
	}

	pi, err := core.LoadInterface(dir)
	if err != nil {
		t.Fatalf("LoadInterface: %v", err)
	}
	return pi
}

func TestBuildAgentEnv_AllKnownKeysPresentAndJSONResolved(t *testing.T) {
	pi := newTestProjectInterface(t)

	env, err := BuildAgentEnv(pi, PIEnvContext{
		InterfaceVersion: "v2.5.0",
		ClientName:       "MEC",
		ClientVersion:    "0.4.0",
		Language:         "zh_cn",
		MaaFWVersion:     "v3.5.1",
		ProjectVersion:   "v1.0.0",
		ControllerName:   "Win32-Window",
		ResourceName:     "CN",
	})
	if err != nil {
		t.Fatalf("BuildAgentEnv failed: %v", err)
	}

	found := map[string]string{}
	for _, kv := range env {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			t.Fatalf("invalid env entry: %q", kv)
		}
		found[kv[:idx]] = kv[idx+1:]
	}

	want := []string{
		"PI_INTERFACE_VERSION",
		"PI_CLIENT_NAME",
		"PI_CLIENT_VERSION",
		"PI_CLIENT_LANGUAGE",
		"PI_CLIENT_MAAFW_VERSION",
		"PI_VERSION",
		"PI_CONTROLLER",
		"PI_RESOURCE",
	}
	for _, k := range want {
		if _, ok := found[k]; !ok {
			t.Fatalf("missing env key: %s", k)
		}
	}

	if v := found["PI_INTERFACE_VERSION"]; v != "v2.5.0" {
		t.Fatalf("PI_INTERFACE_VERSION mismatch: %s", v)
	}
	if v := found["PI_CLIENT_MAAFW_VERSION"]; v != "v3.5.1" {
		t.Fatalf("PI_CLIENT_MAAFW_VERSION mismatch: %s", v)
	}

	var ctrl map[string]interface{}
	if err := json.Unmarshal([]byte(found["PI_CONTROLLER"]), &ctrl); err != nil {
		t.Fatalf("PI_CONTROLLER must be valid JSON: %v; raw=%q", err, found["PI_CONTROLLER"])
	}
	if got := ctrl["label"]; got != "Win32-默认" {
		t.Fatalf("PI_CONTROLLER label should be i18n-resolved, got %v", got)
	}
	if got := ctrl["description"]; got != "默认控制器" {
		t.Fatalf("PI_CONTROLLER description should be i18n-resolved, got %v", got)
	}

	var res map[string]interface{}
	if err := json.Unmarshal([]byte(found["PI_RESOURCE"]), &res); err != nil {
		t.Fatalf("PI_RESOURCE must be valid JSON: %v", err)
	}
	if got := res["label"]; got != "官服" {
		t.Fatalf("PI_RESOURCE label should be i18n-resolved, got %v", got)
	}

	for k, v := range found {
		if strings.ContainsAny(v, "\n\r") {
			t.Fatalf("env %s must not contain newline: %q", k, v)
		}
	}
}

func TestBuildAgentEnv_OmitsEmptyOptionalFields(t *testing.T) {
	pi := newTestProjectInterface(t)

	env, err := BuildAgentEnv(pi, PIEnvContext{
		InterfaceVersion: "v2.5.0",
		Language:         "zh_cn",
	})
	if err != nil {
		t.Fatalf("BuildAgentEnv failed: %v", err)
	}
	for _, kv := range env {
		if strings.HasSuffix(kv, "=") {
			t.Fatalf("env must not include empty-value entry: %q", kv)
		}
	}
}

func TestBuildAgentEnv_UnknownControllerReturnsError(t *testing.T) {
	pi := newTestProjectInterface(t)
	if _, err := BuildAgentEnv(pi, PIEnvContext{
		Language:       "zh_cn",
		ControllerName: "NotExist",
	}); err == nil {
		t.Fatalf("expected error for unknown controller")
	}
}
