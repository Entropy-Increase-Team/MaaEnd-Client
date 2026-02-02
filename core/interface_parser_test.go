package core

import (
	"fmt"
	"testing"
)

func TestLoadInterface(t *testing.T) {
	// 测试加载 MaaEnd v1.6.5 的 interface.json
	maaEndPath := "E:\\Dnyo\\Documents\\work\\endfield\\Endfield-API\\MaaEnd-win-x86_64-v1.6.5"

	pi, err := LoadInterface(maaEndPath)
	if err != nil {
		t.Fatalf("加载 interface.json 失败: %v", err)
	}

	// 验证基本信息
	fmt.Printf("项目名称: %s\n", pi.Name)
	fmt.Printf("项目版本: %s\n", pi.Version)
	fmt.Printf("Import 文件数: %d\n", len(pi.Import))

	// 验证任务是否正确加载
	fmt.Printf("加载的任务数: %d\n", len(pi.Tasks))
	if len(pi.Tasks) == 0 {
		t.Error("任务列表为空，import 可能没有正确工作")
	}

	// 打印所有任务
	fmt.Println("\n=== 任务列表 ===")
	for _, task := range pi.Tasks {
		fmt.Printf("- %s (%s)\n", task.Name, pi.GetI18nString(task.Label, "zh_cn"))
		fmt.Printf("  选项: %v\n", task.Option)
		fmt.Printf("  控制器: %v\n", task.Controller)
	}

	// 验证选项是否正确加载
	fmt.Printf("\n加载的选项数: %d\n", len(pi.Options))
	if len(pi.Options) == 0 {
		t.Error("选项列表为空，import 可能没有正确工作")
	}

	// 打印部分选项
	fmt.Println("\n=== 部分选项（前 5 个） ===")
	count := 0
	for name, opt := range pi.Options {
		if count >= 5 {
			break
		}
		fmt.Printf("- %s (类型: %s)\n", name, opt.Type)
		count++
	}

	// 验证 switch 类型选项
	fmt.Println("\n=== switch 类型选项 ===")
	for name, opt := range pi.Options {
		if opt.Type == "switch" {
			fmt.Printf("- %s: %d cases\n", name, len(opt.Cases))
			if len(opt.Cases) > 0 {
				fmt.Printf("  第一个 case: %s\n", opt.Cases[0].Name)
			}
			break
		}
	}

	// 验证控制器
	fmt.Printf("\n控制器数: %d\n", len(pi.Controllers))
	for _, ctrl := range pi.Controllers {
		fmt.Printf("- %s (类型: %s)\n", ctrl.Name, ctrl.Type)
	}
}
