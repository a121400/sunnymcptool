package tools

import "github.com/a121400/sunnymcptool/mcp"

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "config_get",
		Description: "获取SunnyNet当前配置信息",
		InputSchema: noParamsSchema(),
		Handler:     toolConfigGetHandler,
	})
}

func toolConfigGetHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx().Config
	return map[string]interface{}{
		"success":            true,
		"port":               c.GetPort(),
		"disableUDP":         c.GetDisableUDP(),
		"disableTCP":         c.GetDisableTCP(),
		"disableCache":       c.GetDisableCache(),
		"authentication":     c.GetAuthentication(),
		"globalProxy":        c.GetGlobalProxy(),
		"globalProxyRules":   c.GetGlobalProxyRules(),
		"mustTcpOpen":        c.GetMustTcpOpen(),
		"mustTcpRules":       c.GetMustTcpRules(),
		"certDefault":        c.GetCertDefault(),
		"certCaPath":         c.GetCertCaPath(),
		"certKeyPath":        c.GetCertKeyPath(),
		"replaceRulesCount":  len(c.GetReplaceRules()),
		"hostsRulesCount":    len(c.GetHostsRules()),
		"darkTheme":          c.GetDarkTheme() == 1,
		"requestCertManager": c.GetRequestCertManagerCount(),
	}, nil
}
