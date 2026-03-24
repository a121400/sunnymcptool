package mcp

import (
	"regexp"
	"sync"

	"github.com/a121400/sunnymcptool/MapHash"
)

// HostsRule HOSTS 规则结构
type HostsRule struct {
	Regex  *regexp.Regexp
	Target string
}

// ConfigReplaceRule 配置替换规则
type ConfigReplaceRule struct {
	Type string `json:"Type"`
	Src  string `json:"Src"`
	Dest string `json:"Dest"`
	Hash string `json:"Hash"`
}

// ProxyApp 代理应用接口 - 抽象 SunnyNet 实例
type ProxyApp interface {
	StartProxy() error
	StopProxy()
	GetPort() int
	SetPort(port int)
	GetError() error
	AddProcessName(name string)
	DelProcessName(name string)
}

// AppConfig 应用配置接口
type AppConfig interface {
	GetPort() int
	SetPort(port int)
	GetDisableUDP() bool
	GetDisableTCP() bool
	GetDisableCache() bool
	GetAuthentication() bool
	GetGlobalProxy() string
	GetGlobalProxyRules() string
	GetMustTcpOpen() bool
	GetMustTcpRules() string
	GetCertDefault() bool
	GetCertCaPath() string
	GetCertKeyPath() string
	GetDarkTheme() uint8
	GetReplaceRules() []ConfigReplaceRule
	SetReplaceRules(rules []ConfigReplaceRule)
	GetHostsRules() []ConfigReplaceRule
	SetHostsRules(rules []ConfigReplaceRule)
	GetRequestCertManagerCount() int
	Save() error
}

// HostsRuleManager HOSTS 内部规则管理接口
type HostsRuleManager interface {
	// GetRules 获取当前内部 HOSTS 规则
	GetRules() []HostsRule
	// SetRules 设置内部 HOSTS 规则
	SetRules(rules []HostsRule)
	// AppendRule 追加一条内部 HOSTS 规则
	AppendRule(rule HostsRule)
}

// DataIO 数据导入导出接口
type DataIO interface {
	SaveAllToFile(path string) error
	ImportFromFile(path string) (int, error)
}

// AppContext 应用上下文 - 工具通过此接口访问 main 包的全局变量
type AppContext struct {
	App            ProxyApp
	Config         AppConfig
	HashMap        *MapHash.Map
	TmpLock        *sync.Mutex
	HostsRuleMgr HostsRuleManager
	DataIO       DataIO
}

// 全局应用上下文实例
var GlobalAppContext *AppContext
