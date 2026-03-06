package main

import (
	"github.com/a121400/sunnymcptool/mcp"
	// 导入 tools 包触发 init() 注册
	_ "github.com/a121400/sunnymcptool/mcp/tools"
	"github.com/qtgolang/SunnyNet/src/JsCall"
)

// 全局MCP服务器实例
var mcpServer *mcp.MCPServer

// proxyAppAdapter 将 main 包的 app 适配为 mcp.ProxyApp 接口
type proxyAppAdapter struct{}

func (a *proxyAppAdapter) StartProxy() error {
	if app == nil || app.App == nil {
		return nil
	}
	return app.App.Start().Error
}

func (a *proxyAppAdapter) StopProxy() {
	if app != nil && app.App != nil {
		app.App.Close()
	}
}

func (a *proxyAppAdapter) GetPort() int {
	if app != nil && app.App != nil {
		return app.App.Port()
	}
	return 0
}

func (a *proxyAppAdapter) SetPort(port int) {
	if app != nil && app.App != nil {
		app.App.SetPort(port)
	}
}

func (a *proxyAppAdapter) GetError() error {
	if app != nil && app.App != nil {
		return app.App.Error
	}
	return nil
}

func (a *proxyAppAdapter) AddProcessName(name string) {
	if app != nil && app.App != nil {
		gbkName := JsCall.ToGBK(name)
		app.App.ProcessAddName(gbkName)
	}
}

func (a *proxyAppAdapter) DelProcessName(name string) {
	if app != nil && app.App != nil {
		gbkName := JsCall.ToGBK(name)
		app.App.ProcessDelName(gbkName)
	}
}

// appConfigAdapter 将 GlobalConfig 适配为 mcp.AppConfig 接口
type appConfigAdapter struct{}

func (c *appConfigAdapter) GetPort() int                          { return GlobalConfig.Port }
func (c *appConfigAdapter) SetPort(port int)                      { GlobalConfig.Port = port }
func (c *appConfigAdapter) GetDisableUDP() bool                   { return GlobalConfig.DisableUDP }
func (c *appConfigAdapter) GetDisableTCP() bool                   { return GlobalConfig.DisableTCP }
func (c *appConfigAdapter) GetDisableCache() bool                 { return GlobalConfig.DisableCache }
func (c *appConfigAdapter) GetAuthentication() bool               { return GlobalConfig.Authentication }
func (c *appConfigAdapter) GetGlobalProxy() string                { return GlobalConfig.GlobalProxy }
func (c *appConfigAdapter) GetGlobalProxyRules() string           { return GlobalConfig.GlobalProxyRules }
func (c *appConfigAdapter) GetMustTcpOpen() bool                  { return GlobalConfig.MustTcp.Open }
func (c *appConfigAdapter) GetMustTcpRules() string               { return GlobalConfig.MustTcp.Rules }
func (c *appConfigAdapter) GetCertDefault() bool                  { return GlobalConfig.Cert.Default }
func (c *appConfigAdapter) GetCertCaPath() string                 { return GlobalConfig.Cert.CaPath }
func (c *appConfigAdapter) GetCertKeyPath() string                { return GlobalConfig.Cert.KeyPath }
func (c *appConfigAdapter) GetDarkTheme() uint8                   { return GlobalConfig.DarkTheme }
func (c *appConfigAdapter) GetRequestCertManagerCount() int       { return len(GlobalConfig.RequestCertManager) }
func (c *appConfigAdapter) Save() error                           { return GlobalConfig.saveToFile() }

func (c *appConfigAdapter) GetReplaceRules() []mcp.ConfigReplaceRule {
	rules := make([]mcp.ConfigReplaceRule, len(GlobalConfig.ReplaceRules))
	for i, r := range GlobalConfig.ReplaceRules {
		rules[i] = mcp.ConfigReplaceRule{Type: r.Type, Src: r.Src, Dest: r.Dest, Hash: r.Hash}
	}
	return rules
}

func (c *appConfigAdapter) SetReplaceRules(rules []mcp.ConfigReplaceRule) {
	GlobalConfig.ReplaceRules = make([]ConfigReplaceRules, len(rules))
	for i, r := range rules {
		GlobalConfig.ReplaceRules[i] = ConfigReplaceRules{Type: r.Type, Src: r.Src, Dest: r.Dest, Hash: r.Hash}
	}
}

func (c *appConfigAdapter) GetHostsRules() []mcp.ConfigReplaceRule {
	rules := make([]mcp.ConfigReplaceRule, len(GlobalConfig.HostsRules))
	for i, r := range GlobalConfig.HostsRules {
		rules[i] = mcp.ConfigReplaceRule{Type: r.Type, Src: r.Src, Dest: r.Dest, Hash: r.Hash}
	}
	return rules
}

func (c *appConfigAdapter) SetHostsRules(rules []mcp.ConfigReplaceRule) {
	GlobalConfig.HostsRules = make([]ConfigReplaceRules, len(rules))
	for i, r := range rules {
		GlobalConfig.HostsRules[i] = ConfigReplaceRules{Type: r.Type, Src: r.Src, Dest: r.Dest, Hash: r.Hash}
	}
}

// cryptoAnalyzerAdapter 将 cryptoAnalyzer 适配为 mcp.CryptoAnalyzer 接口
type cryptoAnalyzerAdapter struct{}

func (a *cryptoAnalyzerAdapter) ParsePacket(data []byte) (header interface{}, rawHex, payloadHex, decryptedHex, protobufTree string, err error) {
	if cryptoAnalyzer == nil {
		return nil, "", "", "", "", nil
	}
	result, e := cryptoAnalyzer.ParsePacket(data)
	if e != nil {
		return nil, "", "", "", "", e
	}
	return result.Header, result.RawHex, result.PayloadHex, result.DecryptedHex, result.ProtobufTree, nil
}

func (a *cryptoAnalyzerAdapter) ParseProtobuf(data []byte, skip int) string {
	if cryptoAnalyzer == nil {
		return ""
	}
	return cryptoAnalyzer.ParseProtobuf(data, skip)
}

func (a *cryptoAnalyzerAdapter) GetCurrentConfigName() string {
	if cryptoAnalyzer == nil {
		return ""
	}
	config := cryptoAnalyzer.GetCurrentConfig()
	if config == nil {
		return ""
	}
	return config.Name
}

func (a *cryptoAnalyzerAdapter) GetCurrentConfig() (name, aesKey, aesIV string, headerSize int, msgNames map[int]string, ok bool) {
	if cryptoAnalyzer == nil {
		return "", "", "", 0, nil, false
	}
	config := cryptoAnalyzer.GetCurrentConfig()
	if config == nil {
		return "", "", "", 0, nil, false
	}
	return config.Name, config.AESKey, config.AESIV, config.HeaderSize, config.MsgNames, true
}

func (a *cryptoAnalyzerAdapter) GetAllConfigs() []mcp.CryptoConfigInfo {
	if cryptoAnalyzer == nil {
		return nil
	}
	configs := cryptoAnalyzer.GetAllConfigs()
	currentConfig := cryptoAnalyzer.GetCurrentConfig()
	currentName := ""
	if currentConfig != nil {
		currentName = currentConfig.Name
	}
	result := make([]mcp.CryptoConfigInfo, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, mcp.CryptoConfigInfo{
			Name: cfg.Name, AESKey: cfg.AESKey, AESIV: cfg.AESIV,
			HeaderSize: cfg.HeaderSize, IsCurrent: cfg.Name == currentName,
		})
	}
	return result
}

func (a *cryptoAnalyzerAdapter) AddConfig(name, aesKey, aesIV string, headerSize int) {
	if cryptoAnalyzer == nil {
		return
	}
	config := &CryptoConfig{
		Name: name, AESKey: aesKey, AESIV: aesIV,
		HeaderSize: headerSize, MsgNames: make(map[int]string),
	}
	cryptoAnalyzer.AddConfig(config)
}

func (a *cryptoAnalyzerAdapter) SetCurrentConfig(name string) error {
	if cryptoAnalyzer == nil {
		return nil
	}
	return cryptoAnalyzer.SetCurrentConfig(name)
}

// InitMCPContext 初始化 MCP 应用上下文
// 必须在 app 和 GlobalConfig 初始化之后调用
func InitMCPContext() {
	mcp.GlobalAppContext = &mcp.AppContext{
		App:            &proxyAppAdapter{},
		Config:         &appConfigAdapter{},
		HashMap:        HashMap,
		TmpLock:        &_TmpLock,
		HostsRuleMgr:   &hostsRuleManagerAdapter{},
		CryptoAnalyzer: &cryptoAnalyzerAdapter{},
	}
}

// hostsRuleManagerAdapter 适配 main 包的 _HostsRules 到 mcp.HostsRuleManager 接口
type hostsRuleManagerAdapter struct{}

func (a *hostsRuleManagerAdapter) GetRules() []mcp.HostsRule {
	rules := make([]mcp.HostsRule, len(_HostsRules))
	for i, r := range _HostsRules {
		rules[i] = mcp.HostsRule{Regex: r.regex, Target: r.target}
	}
	return rules
}

func (a *hostsRuleManagerAdapter) SetRules(rules []mcp.HostsRule) {
	newRules := make([]HostsRules, len(rules))
	for i, r := range rules {
		newRules[i] = HostsRules{regex: r.Regex, target: r.Target}
	}
	_HostsRules = newRules
}

func (a *hostsRuleManagerAdapter) AppendRule(rule mcp.HostsRule) {
	_HostsRules = append(_HostsRules, HostsRules{regex: rule.Regex, target: rule.Target})
}
