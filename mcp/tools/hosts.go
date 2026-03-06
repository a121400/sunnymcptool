package tools

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "hosts_list",
		Description: "列出所有HOSTS规则，返回每条规则的源地址、目标地址和唯一标识",
		InputSchema: noParamsSchema(),
		Handler:     toolHostsListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "hosts_add",
		Description: "添加HOSTS规则，源地址为正则模式，目标地址为替换目标。添加后立即生效",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"source": map[string]interface{}{
					"type":        "string",
					"description": "源地址正则模式（用于匹配域名）",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "目标地址（替换为的地址）",
				},
			},
			"required": []string{"source", "target"},
		},
		Handler: toolHostsAddHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "hosts_remove",
		Description: "删除指定的HOSTS规则，删除后立即生效",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"index": map[string]interface{}{
					"type":        "integer",
					"description": "规则索引（从0开始，通过hosts_list获取）",
				},
			},
			"required": []string{"index"},
		},
		Handler: toolHostsRemoveHandler,
	})
}

func toolHostsListHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	rules := c.Config.GetHostsRules()
	if rules == nil {
		rules = []mcp.ConfigReplaceRule{}
	}

	ruleList := make([]map[string]interface{}, 0, len(rules))
	for i, rule := range rules {
		ruleList = append(ruleList, map[string]interface{}{
			"index":  i,
			"source": rule.Src,
			"target": rule.Dest,
			"hash":   rule.Hash,
		})
	}

	return map[string]interface{}{
		"success": true,
		"rules":   ruleList,
		"total":   len(ruleList),
	}, nil
}

func toolHostsAddHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	source := v.RequireString("source")
	target := v.RequireString("target")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if source == "" {
		return nil, fmt.Errorf("源地址不能为空")
	}
	if target == "" {
		return nil, fmt.Errorf("目标地址不能为空")
	}

	// 预处理源地址
	processedSource := source
	processedSource = strings.ReplaceAll(processedSource, "\\\\", "\\")
	processedSource = strings.ReplaceAll(processedSource, ".*", "{点星}")
	processedSource = strings.ReplaceAll(processedSource, "*", ".*")
	processedSource = strings.ReplaceAll(processedSource, "{点星}", ".*")

	regex, err := regexp.Compile(processedSource)
	if err != nil {
		return nil, fmt.Errorf("源地址正则模式无效: %s", err.Error())
	}

	hash := fmt.Sprintf("%d", time.Now().UnixNano())
	processedTarget := strings.ReplaceAll(target, "\\\\", "\\")

	c := ctx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	// 添加到配置
	configRule := mcp.ConfigReplaceRule{
		Hash: hash,
		Src:  processedSource,
		Dest: processedTarget,
	}
	rules := c.Config.GetHostsRules()
	rules = append(rules, configRule)
	c.Config.SetHostsRules(rules)
	_ = c.Config.Save()

	// 立即更新内部规则
	c.HostsRuleMgr.AppendRule(mcp.HostsRule{
		Regex:  regex,
		Target: processedTarget,
	})

	return map[string]interface{}{
		"success": true,
		"rule": map[string]interface{}{
			"index":  len(rules) - 1,
			"source": processedSource,
			"target": processedTarget,
			"hash":   hash,
		},
		"message": "HOSTS规则已添加并立即生效",
	}, nil
}

func toolHostsRemoveHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	index := v.RequireInt("index")
	if err := v.Error(); err != nil {
		return nil, err
	}

	c := ctx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	rules := c.Config.GetHostsRules()
	if index < 0 || index >= len(rules) {
		return nil, fmt.Errorf("索引 %d 超出范围，当前共有 %d 条规则（索引范围: 0-%d）",
			index, len(rules), len(rules)-1)
	}

	removedRule := rules[index]
	rules = append(rules[:index], rules[index+1:]...)
	c.Config.SetHostsRules(rules)
	_ = c.Config.Save()

	// 重建内部规则
	reloadHostsRulesFromConfig(c)

	return map[string]interface{}{
		"success": true,
		"removed": map[string]interface{}{
			"source": removedRule.Src,
			"target": removedRule.Dest,
			"hash":   removedRule.Hash,
		},
		"message": "HOSTS规则已删除并立即生效",
	}, nil
}

// reloadHostsRulesFromConfig 从配置重建内部 HOSTS 规则
// 调用前需已获取 TmpLock
func reloadHostsRulesFromConfig(c *mcp.AppContext) {
	var newRules []mcp.HostsRule
	for _, configRule := range c.Config.GetHostsRules() {
		regex, err := regexp.Compile(configRule.Src)
		if err != nil {
			continue
		}
		newRules = append(newRules, mcp.HostsRule{
			Regex:  regex,
			Target: configRule.Dest,
		})
	}
	c.HostsRuleMgr.SetRules(newRules)
}
