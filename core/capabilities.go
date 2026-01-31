package core

import (
	"maaend-client/client"
)

// CapabilitiesBuilder 设备能力构建器
type CapabilitiesBuilder struct {
	pi   *ProjectInterface
	lang string
}

// NewCapabilitiesBuilder 创建能力构建器
func NewCapabilitiesBuilder(pi *ProjectInterface, lang string) *CapabilitiesBuilder {
	if lang == "" {
		lang = "zh_cn"
	}
	return &CapabilitiesBuilder{
		pi:   pi,
		lang: lang,
	}
}

// Build 构建设备能力
func (b *CapabilitiesBuilder) Build() *client.CapabilitiesPayload {
	capabilities := &client.CapabilitiesPayload{
		Controllers: b.pi.GetControllerNames(),
		Resources:   b.pi.GetResourceNames(),
		Tasks:       make([]client.TaskInfo, 0, len(b.pi.Tasks)),
	}

	// 构建任务信息
	for _, task := range b.pi.Tasks {
		taskInfo := client.TaskInfo{
			Name:        task.Name,
			Label:       b.pi.GetI18nString(task.Label, b.lang),
			Description: b.pi.GetI18nString(task.Description, b.lang),
			Options:     b.buildTaskOptions(task.Option),
		}
		capabilities.Tasks = append(capabilities.Tasks, taskInfo)
	}

	return capabilities
}

// buildTaskOptions 构建任务选项
func (b *CapabilitiesBuilder) buildTaskOptions(optionNames []string) []client.OptionInfo {
	var options []client.OptionInfo

	for _, optName := range optionNames {
		opt := b.pi.GetOption(optName)
		if opt == nil {
			continue
		}

		optInfo := client.OptionInfo{
			Name:        optName,
			Type:        opt.Type,
			Label:       b.pi.GetI18nString(opt.Label, b.lang),
			DefaultCase: opt.DefaultCase,
		}

		// 构建 cases
		if len(opt.Cases) > 0 {
			optInfo.Cases = make([]client.CaseInfo, 0, len(opt.Cases))
			for _, c := range opt.Cases {
				caseInfo := client.CaseInfo{
					Name:  c.Name,
					Label: b.pi.GetI18nString(c.Label, b.lang),
				}
				optInfo.Cases = append(optInfo.Cases, caseInfo)
			}
		}

		options = append(options, optInfo)
	}

	return options
}
