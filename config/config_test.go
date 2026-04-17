package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAndLoadConfig 在临时目录里写入指定 YAML，加载并返回 (cfg, configPath)。
// 测试结束时自动清理全局状态，避免用例之间污染。
func writeAndLoadConfig(t *testing.T, yaml string) (*Config, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	t.Cleanup(func() {
		globalConfig = nil
		configFilePath = ""
	})

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg, path
}

func TestLoad_OldVersionOverwrittenInMemory(t *testing.T) {
	cfg, _ := writeAndLoadConfig(t, `# MaaEnd Client 配置文件
version: "0.1.0"
server:
  ws_url: "ws://localhost:15618/ws/maaend"
maaend:
  path: ""
device:
  name: "test-device"
  token: ""
logging:
  level: "info"
  file: ""
`)

	if cfg.Version != CurrentVersion {
		t.Fatalf("expect cfg.Version == %q, got %q", CurrentVersion, cfg.Version)
	}
}

func TestLoad_EmptyVersionTakesCurrent(t *testing.T) {
	cfg, _ := writeAndLoadConfig(t, `# MaaEnd Client 配置文件
server:
  ws_url: "ws://localhost:15618/ws/maaend"
maaend:
  path: ""
device:
  name: "test-device"
  token: ""
logging:
  level: "info"
  file: ""
`)

	if cfg.Version != CurrentVersion {
		t.Fatalf("expect cfg.Version == %q when file omits version, got %q", CurrentVersion, cfg.Version)
	}
}

func TestEnsureConfigFormat_RewritesWhenVersionMismatches(t *testing.T) {
	_, path := writeAndLoadConfig(t, `# MaaEnd Client 配置文件
version: "0.1.0"
server:
  ws_url: "ws://localhost:15618/ws/maaend"
maaend:
  path: ""
device:
  name: "test-device"
  token: ""
logging:
  level: "info"
  file: ""
`)

	// 文件应该仍是旧版本（只有 Load 阶段内存里被改了）
	old, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(old), `version: "0.1.0"`) {
		t.Fatalf("precondition failed: file should still contain old version\ngot:\n%s", old)
	}

	if err := EnsureConfigFormat(); err != nil {
		t.Fatalf("EnsureConfigFormat: %v", err)
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	txt := string(updated)

	wantLine := `version: "` + CurrentVersion + `"`
	if !strings.Contains(txt, wantLine) {
		t.Fatalf("file should contain %q after EnsureConfigFormat, got:\n%s", wantLine, txt)
	}
	if strings.Contains(txt, `version: "0.1.0"`) {
		t.Fatalf("file should no longer contain old version, got:\n%s", txt)
	}
}

func TestEnsureConfigFormat_NoRewriteWhenVersionMatches(t *testing.T) {
	_, path := writeAndLoadConfig(t, `# MaaEnd Client 配置文件

# 客户端版本号
version: "`+CurrentVersion+`"

server:
  ws_url: "ws://localhost:15618/ws/maaend"
  connect_timeout: 10s
  heartbeat_interval: 30s
  reconnect_max_delay: 30s

maaend:
  path: ""
  win32_class_regex: ""
  win32_window_regex: ""

device:
  name: "test-device"
  token: ""

logging:
  level: "info"
  file: ""
`)

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	originalModTime := mustStat(t, path).ModTime()

	if err := EnsureConfigFormat(); err != nil {
		t.Fatalf("EnsureConfigFormat: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}

	if !equalIgnoreBOM(original, after) {
		// 这里不强制要求字节完全一致（可能因为未来逻辑原因需要规范化），
		// 但至少对于当前测试场景，同版本应不触发改写。
		newModTime := mustStat(t, path).ModTime()
		if !newModTime.Equal(originalModTime) {
			t.Fatalf("EnsureConfigFormat rewrote the file for matching version")
		}
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	return info
}

func equalIgnoreBOM(a, b []byte) bool {
	bom := []byte{0xEF, 0xBB, 0xBF}
	trim := func(x []byte) []byte {
		if len(x) >= 3 && x[0] == bom[0] && x[1] == bom[1] && x[2] == bom[2] {
			return x[3:]
		}
		return x
	}
	return string(trim(a)) == string(trim(b))
}
