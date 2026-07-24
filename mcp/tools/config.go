package tools

import (
	"errors"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"
	stls "github.com/qtgolang/SunnyNet/src/crypto/tls"
	"github.com/qtgolang/SunnyNet/src/httpClient"
	"github.com/qtgolang/SunnyNet/src/tlsClient/tlsClient/profiles"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "config_get",
		Description: "获取SunnyNet当前配置信息",
		InputSchema: noParamsSchema(),
		Handler:     toolConfigGetHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "tls_fingerprint_toggle",
		Description: "切换TLS指纹模式：开启后重发请求使用指定浏览器TLS指纹(如chrome_120)绕过WAF",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enabled": map[string]interface{}{"type": "boolean", "description": "是否启用TLS指纹"},
				"profile": map[string]interface{}{"type": "string", "description": "TLS指纹名称，如 chrome_120, chrome_124, firefox_120 等"},
			},
			"required": []string{},
		},
		Handler: toolTlsFingerprintToggleHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "tls_fingerprint_status",
		Description: "获取当前TLS指纹模式状态",
		InputSchema: noParamsSchema(),
		Handler:     toolTlsFingerprintStatusHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "tls_fingerprint_profiles",
		Description: "列出所有可用的TLS指纹profile名称",
		InputSchema: noParamsSchema(),
		Handler:     toolTlsFingerprintProfilesHandler,
	})
}

var helloIDMap = map[string]stls.ClientHelloID{
	"chrome_120":  stls.HelloChrome_120,
	"chrome_124":  stls.HelloChrome_Auto,
	"firefox_120": stls.HelloFirefox_120,
}

func toolTlsFingerprintToggleHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	if enabled, ok := args["enabled"]; ok {
		if b, ok2 := enabled.(bool); ok2 {
			MapHash.UseTlsFingerprint = b
			httpClient.UseTlsFingerprint = b
		}
	}
	if profileName, ok := args["profile"]; ok {
		if name, ok2 := profileName.(string); ok2 && name != "" {
			if _, exists := profiles.MappedTLSClients[name]; exists {
				MapHash.TlsProfileName = name
				if hid, ok3 := helloIDMap[name]; ok3 {
					httpClient.TlsHelloID = hid
				} else {
					httpClient.TlsHelloID = stls.HelloChrome_120
				}
			} else {
				return map[string]interface{}{
					"success": false,
					"error":   "未知的TLS profile: " + name,
				}, nil
			}
		}
	}
	_ = v
	return map[string]interface{}{
		"success": true,
		"enabled": MapHash.UseTlsFingerprint,
		"profile": MapHash.TlsProfileName,
		"message": func() string {
			if MapHash.UseTlsFingerprint {
				return "TLS指纹已开启，重发请求将使用 " + MapHash.TlsProfileName + " 指纹"
			}
			return "TLS指纹已关闭，重发请求使用默认方式"
		}(),
	}, nil
}

func toolTlsFingerprintStatusHandler(args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"success": true,
		"enabled": MapHash.UseTlsFingerprint,
		"profile": MapHash.TlsProfileName,
	}, nil
}

func toolTlsFingerprintProfilesHandler(args map[string]interface{}) (interface{}, error) {
	names := make([]string, 0, len(profiles.MappedTLSClients))
	for name := range profiles.MappedTLSClients {
		names = append(names, name)
	}
	return map[string]interface{}{
		"success":  true,
		"profiles": names,
		"current":  MapHash.TlsProfileName,
		"enabled":  MapHash.UseTlsFingerprint,
	}, nil
}

func toolConfigGetHandler(args map[string]interface{}) (interface{}, error) {
	appCtx := ctx()
	if appCtx == nil || appCtx.Config == nil {
		return nil, errors.New("应用上下文未初始化")
	}
	c := appCtx.Config
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
