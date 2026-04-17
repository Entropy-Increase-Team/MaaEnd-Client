package maa

import (
	"fmt"

	"maaend-client/core"
)

// PIEnvContext 构造 PI_* 环境变量所需的 Client 侧上下文。
// 参考 MaaFW PI V2.5.0「Agent 子进程环境变量」约定。
type PIEnvContext struct {
	InterfaceVersion string // MEC 实现的 PI 扩展协议版本，如 "v2.5.0"
	ClientName       string // Client 标识，如 "MEC"
	ClientVersion    string // Client 自身版本
	Language         string // 当前 UI 语言（zh_cn 等）
	MaaFWVersion     string // 集成的 MaaFramework 库版本
	ProjectVersion   string // interface.json 顶层 version 字段
	ControllerName   string // 当前选中的 controller name
	ResourceName     string // 当前选中的 resource name
}

// BuildAgentEnv 按照 PI v2.5.0 约定构造 Agent 子进程环境变量列表。
// 返回值格式为 []string{"KEY=VALUE", ...}，便于直接赋给 exec.Cmd.Env。
//
// 对不可用的项按协议规定为空或不设置；PI_CONTROLLER / PI_RESOURCE 为单行紧凑 JSON，
// i18n 字段已解析为展示文本。
func BuildAgentEnv(pi *core.ProjectInterface, ctx PIEnvContext) ([]string, error) {
	env := make([]string, 0, 8)

	set := func(k, v string) {
		if v == "" {
			return
		}
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	set("PI_INTERFACE_VERSION", ctx.InterfaceVersion)
	set("PI_CLIENT_NAME", ctx.ClientName)
	set("PI_CLIENT_VERSION", ctx.ClientVersion)
	set("PI_CLIENT_LANGUAGE", ctx.Language)
	set("PI_CLIENT_MAAFW_VERSION", ctx.MaaFWVersion)
	set("PI_VERSION", ctx.ProjectVersion)

	if ctx.ControllerName != "" {
		ctrlJSON, err := core.ResolveControllerForEnv(pi, ctx.ControllerName, ctx.Language)
		if err != nil {
			return nil, fmt.Errorf("构造 PI_CONTROLLER 失败: %w", err)
		}
		set("PI_CONTROLLER", ctrlJSON)
	}

	if ctx.ResourceName != "" {
		resJSON, err := core.ResolveResourceForEnv(pi, ctx.ResourceName, ctx.Language)
		if err != nil {
			return nil, fmt.Errorf("构造 PI_RESOURCE 失败: %w", err)
		}
		set("PI_RESOURCE", resJSON)
	}

	return env, nil
}
