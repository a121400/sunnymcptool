package tools

import (
	"errors"
	"fmt"

	"github.com/a121400/sunnymcptool/CommAnd"
	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "process_list",
		Description: "获取当前系统运行的进程列表（用于选择要拦截的进程）",
		InputSchema: noParamsSchema(),
		Handler:     toolProcessListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "process_add_name",
		Description: "添加要拦截的进程名（Windows需要加载驱动）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "进程名称（如：chrome.exe）",
				},
			},
			"required": []string{"name"},
		},
		Handler: toolProcessAddNameHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "process_remove_name",
		Description: "移除已添加的拦截进程名",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "进程名称（如：chrome.exe）",
				},
			},
			"required": []string{"name"},
		},
		Handler: toolProcessRemoveNameHandler,
	})
}

func toolProcessListHandler(args map[string]interface{}) (interface{}, error) {
	processes := CommAnd.EnumerateProcesses()
	return map[string]interface{}{
		"success":   true,
		"processes": processes,
	}, nil
}

func toolProcessAddNameHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	name := v.RequireString("name")
	if err := v.Error(); err != nil {
		return nil, err
	}
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	c.App.AddProcessName(name)
	return map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("已添加进程名: %s", name),
	}, nil
}

func toolProcessRemoveNameHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	name := v.RequireString("name")
	if err := v.Error(); err != nil {
		return nil, err
	}
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	c.App.DelProcessName(name)
	return map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("已移除进程名: %s", name),
	}, nil
}
