package tools

import (
	"errors"
	"fmt"

	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_start",
		Description: "启动SunnyNet代理服务",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyStartHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_stop",
		Description: "停止SunnyNet代理服务",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyStopHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_set_port",
		Description: "设置代理端口号",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "代理端口号 (1-65535)",
					"minimum":     1,
					"maximum":     65535,
				},
			},
			"required": []string{"port"},
		},
		Handler: toolProxySetPortHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_get_status",
		Description: "获取代理服务状态，包括运行状态、端口号等信息",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyGetStatusHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_set_ie",
		Description: "设置系统IE代理（将流量导入SunnyNet）",
		InputSchema: noParamsSchema(),
		Handler:     toolProxySetIEHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_unset_ie",
		Description: "取消系统IE代理",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyUnsetIEHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_pause_capture",
		Description: "暂停捕获（停止记录新的请求）",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyPauseCaptureHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "proxy_resume_capture",
		Description: "恢复捕获（继续记录新的请求）",
		InputSchema: noParamsSchema(),
		Handler:     toolProxyResumeCaptureHandler,
	})
}

func toolProxyStartHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	err := c.App.StartProxy()
	if err != nil {
		return nil, fmt.Errorf("启动代理服务失败: %s", err.Error())
	}
	return map[string]interface{}{
		"success": true,
		"port":    c.App.GetPort(),
		"message": "代理服务已启动",
	}, nil
}

func toolProxyStopHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	c.App.StopProxy()
	return map[string]interface{}{
		"success": true,
		"message": "代理服务已停止",
	}, nil
}

func toolProxySetPortHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	port := v.RequireInt("port")
	if err := v.Error(); err != nil {
		return nil, err
	}
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	if port < 1 || port > 65535 {
		return nil, errors.New("端口号必须在 1-65535 之间")
	}
	c.App.StopProxy()
	c.App.SetPort(port)
	err := c.App.StartProxy()
	if err != nil {
		return nil, fmt.Errorf("设置端口后启动代理服务失败: %s", err.Error())
	}
	c.Config.SetPort(port)
	_ = c.Config.Save()
	if c.NotifyUI != nil {
		c.NotifyUI("MCP修改端口", map[string]interface{}{"port": port})
	}
	return map[string]interface{}{
		"success": true,
		"port":    port,
		"message": fmt.Sprintf("端口已设置为 %d", port),
	}, nil
}

func toolProxyGetStatusHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil {
		return map[string]interface{}{
			"running": false,
			"error":   "应用上下文未初始化",
		}, nil
	}
	if c.App == nil {
		return map[string]interface{}{
			"running":        false,
			"port":           c.Config.GetPort(),
			"error":          "SunnyNet实例未初始化",
			"disableUDP":     c.Config.GetDisableUDP(),
			"disableTCP":     c.Config.GetDisableTCP(),
			"disableCache":   c.Config.GetDisableCache(),
			"authentication": c.Config.GetAuthentication(),
		}, nil
	}
	appErr := c.App.GetError()
	errStr := ""
	if appErr != nil {
		errStr = appErr.Error()
	}
	capturing := true
	if c.GetCapturing != nil {
		capturing = c.GetCapturing()
	}
	return map[string]interface{}{
		"running":        errStr == "",
		"port":           c.App.GetPort(),
		"error":          errStr,
		"disableUDP":     c.Config.GetDisableUDP(),
		"disableTCP":     c.Config.GetDisableTCP(),
		"disableCache":   c.Config.GetDisableCache(),
		"authentication": c.Config.GetAuthentication(),
		"globalProxy":    c.Config.GetGlobalProxy(),
		"capturing":      capturing,
	}, nil
}

func toolProxySetIEHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	ok := c.App.SetIEProxy()
	if !ok {
		return nil, errors.New("设置IE代理失败")
	}
	if c.NotifyUI != nil {
		c.NotifyUI("MCP设置IE代理状态", map[string]interface{}{"state": true})
	}
	return map[string]interface{}{
		"success": true,
		"message": "已设置系统IE代理",
	}, nil
}

func toolProxyUnsetIEHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}
	ok := c.App.CancelIEProxy()
	if !ok {
		return nil, errors.New("取消IE代理失败")
	}
	if c.NotifyUI != nil {
		c.NotifyUI("MCP设置IE代理状态", map[string]interface{}{"state": false})
	}
	return map[string]interface{}{
		"success": true,
		"message": "已取消系统IE代理",
	}, nil
}

func toolProxyPauseCaptureHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil {
		return nil, errors.New("应用上下文未初始化")
	}
	if c.SetCapturing == nil {
		return nil, errors.New("捕获控制未初始化")
	}
	c.SetCapturing(false)
	if c.NotifyUI != nil {
		c.NotifyUI("MCP设置捕获状态", map[string]interface{}{"state": false})
	}
	return map[string]interface{}{
		"success":   true,
		"capturing": false,
		"message":   "已暂停捕获",
	}, nil
}

func toolProxyResumeCaptureHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil {
		return nil, errors.New("应用上下文未初始化")
	}
	if c.SetCapturing == nil {
		return nil, errors.New("捕获控制未初始化")
	}
	c.SetCapturing(true)
	if c.NotifyUI != nil {
		c.NotifyUI("MCP设置捕获状态", map[string]interface{}{"state": true})
	}
	return map[string]interface{}{
		"success":   true,
		"capturing": true,
		"message":   "已恢复捕获",
	}, nil
}
