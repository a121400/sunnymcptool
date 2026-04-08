package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"
	_ "github.com/a121400/sunnymcptool/mcp/tools"
	"github.com/qtgolang/SunnyNet/src/JsCall"
)

// 全局MCP服务器实例
var mcpServer *mcp.MCPServer
var mcpStartError string

// proxyAppAdapter 将 main 包的 app 适配为 mcp.ProxyApp 接口
type proxyAppAdapter struct{}

func (a *proxyAppAdapter) StartProxy() error {
	if app == nil || app.App == nil {
		return fmt.Errorf("SunnyNet实例未初始化")
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

func (a *proxyAppAdapter) SetIEProxy() bool {
	if app != nil && app.App != nil {
		return app.App.SetIEProxy()
	}
	return false
}

func (a *proxyAppAdapter) CancelIEProxy() bool {
	if app != nil && app.App != nil {
		return app.App.CancelIEProxy()
	}
	return false
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
	RebuildReplaceRulesFromConfig()
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

// dataIOAdapter 数据导入导出适配器
type dataIOAdapter struct{}

func (d *dataIOAdapter) SaveAllToFile(path string) error {
	if !strings.HasSuffix(strings.ToLower(path), ".syn") {
		path += ".syn"
	}
	ok := HashMap.SaveToFile(path, true, nil, func(s string) {})
	if !ok {
		return fmt.Errorf("保存失败")
	}
	return nil
}

func (d *dataIOAdapter) ImportFromFile(path string) (int, error) {
	bs, e := os.ReadFile(path)
	if e != nil {
		return 0, fmt.Errorf("读取文件失败: %v", e)
	}
	DATA := MapHash.BrUnCompress(bs)
	var openData []*MapHash.Request
	e = json.Unmarshal(DATA, &openData)
	if e != nil {
		return 0, fmt.Errorf("解析文件失败: %v", e)
	}
	var listInfo []ListInfo
	for _, v := range openData {
		if v == nil {
			continue
		}
		theology := HashMap.CreateUniqueID()
		HashMap.SetRequest(theology, v)
		state := strconv.Itoa(v.Response.StateCode)
		if !strings.Contains(strings.ToUpper(v.URL), "HTTP") {
			state = "已断开"
		}
		responseType := ""
		method := v.Method
		ico := "websocket_close"
		responseLen := ""
		if v.Way == "Websocket" {
			method = "Websocket"
			responseType = "Websocket"
			responseLen = strconv.Itoa(v.SendNum) + "/" + strconv.Itoa(v.RecNum)
		} else if v.Way == "UDP" {
			method = "UDP"
			responseType = "UDP"
			responseLen = strconv.Itoa(v.SendNum) + "/" + strconv.Itoa(v.RecNum)
		} else if strings.Contains(strings.ToUpper(method), "TCP") {
			responseLen = strconv.Itoa(v.SendNum) + "/" + strconv.Itoa(v.RecNum)
			responseType = method
		} else {
			if v.Response.Header != nil {
				a := v.Response.Header["Content-Type"]
				if len(a) > 0 {
					responseType = a[0]
				} else {
					a = v.Response.Header["content-type"]
					if len(a) > 0 {
						responseType = a[0]
					}
				}
				if responseType != "" {
					arr := strings.Split(responseType+";", ";")
					if len(arr) > 0 {
						responseType = arr[0]
					}
				}
			}
			ico = UpdateIco(v, responseType)
			responseLen = strconv.Itoa(len(v.Response.Body))
		}
		tmp := ListInfo{
			MessageId: -1,
			Theology:  theology,
			State:     state,
			URL:       v.URL,
			ClientIP:  v.ClientIP,
			PID:       v.PID,
			Method:    method,
			Ico:       ico,
			Len:       responseLen,
			Type:      responseType,
			SendTime:  v.SendTime,
			RecTime:   v.RecTime,
			Notes:     v.Notes,
		}
		tmp.Color.TagColor = v.Color.TagColor
		tmp.Color.Search = v.Color.Search
		listInfo = append(listInfo, tmp)
	}
	if len(listInfo) > 0 {
		CallJs("插入列表", listInfo)
	}
	return len(openData), nil
}

// InitMCPContext 初始化 MCP 应用上下文
func InitMCPContext() {
	mcp.GlobalAppContext = &mcp.AppContext{
		App:          &proxyAppAdapter{},
		Config:       &appConfigAdapter{},
		HashMap:      HashMap,
		TmpLock:      &_TmpLock,
		HostsRuleMgr: &hostsRuleManagerAdapter{},
		DataIO:       &dataIOAdapter{},
		SetCapturing: SetWorkingState,
		GetCapturing: GetWorkingState,
		SearchFunc: func(keyword, searchType, color string) interface{} {
			fv := &FindValue{
				Value:   keyword,
				Type:    searchType,
				Color:   color,
				Range:   "全部",
				Options: "取消之前的颜色标记",
			}
			result := fv.Find()
			if sr, ok := result.(*SearchResult); ok && sr != nil {
				CallJs("MCP搜索高亮", sr)
			}
			return result
		},
		CancelSearchFunc: func() []int {
			cleared := CancelSearch()
			CallJs("MCP取消搜索高亮", cleared)
			return cleared
		},
		NotifyUI: func(event string, data interface{}) {
			CallJs(event, data)
		},
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
