package tools

import (
	"errors"
	"fmt"
	"time"

	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "replace_rules_list",
		Description: "列出当前所有替换规则，返回每条规则的源地址、目标地址和唯一标识",
		InputSchema: noParamsSchema(),
		Handler:     toolReplaceRulesListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "replace_rules_add",
		Description: "添加替换规则。类型：Base64、HEX、String(UTF8)、String(GBK) 为全局字节替换；仅URL 只改 URL；请求体/响应体 只改对应正文；请求头/响应头 的 source 填头名（整段赋值，target 为空则删除）或 头名||片段；正则请求/正则响应 的 source 为正则，target 支持 $1；响应文件 的 source 为 URL 中要出现的文本，target 为本地文件路径，每次请求重新读文件。url 可选，只对 URL 包含该文本的请求生效，以 re: 开头则按正则匹配 URL。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type":        "string",
					"description": "替换类型",
					"enum":        []string{"Base64", "HEX", "String(UTF8)", "String(GBK)", "响应文件", "请求头", "响应头", "请求体", "响应体", "正则请求", "正则响应", "仅URL"},
					"default":     "String(UTF8)",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "源内容。请求头/响应头填头名或 头名||要替换的片段；正则为表达式；响应文件为 URL 中包含的文件名或路径片段",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "替换内容。请求头/响应头为新的完整值，留空表示删除该头；响应文件为本地绝对路径",
					"default":     "",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "可选。只对 URL 包含这段文字的请求生效。以 re: 开头则按正则匹配完整 URL。留空表示全部流量",
					"default":     "",
				},
			},
			"required": []string{"source"},
		},
		Handler: toolReplaceRulesAddHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "replace_rules_remove",
		Description: "删除指定的替换规则",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"hash": map[string]interface{}{
					"type":        "string",
					"description": "规则的唯一标识Hash",
				},
				"index": map[string]interface{}{
					"type":        "integer",
					"description": "规则索引（从0开始，通过replace_rules_list获取）",
				},
			},
		},
		Handler: toolReplaceRulesRemoveHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "replace_rules_clear",
		Description: "清空所有替换规则",
		InputSchema: noParamsSchema(),
		Handler:     toolReplaceRulesClearHandler,
	})
}

func toolReplaceRulesListHandler(args map[string]interface{}) (interface{}, error) {
	c, err := requireCtx()
	if err != nil || c.Config == nil {
		return nil, errCtxNil
	}
	rules := c.Config.GetReplaceRules()
	if rules == nil {
		rules = []mcp.ConfigReplaceRule{}
	}
	items := make([]map[string]interface{}, len(rules))
	for i, r := range rules {
		items[i] = map[string]interface{}{
			"index": i,
			"type":  r.Type,
			"src":   r.Src,
			"dest":  r.Dest,
			"hash":  r.Hash,
			"url":   r.Scope,
		}
	}
	return map[string]interface{}{
		"success": true,
		"rules":   items,
		"total":   len(rules),
	}, nil
}

func toolReplaceRulesAddHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	source := v.RequireString("source")
	ruleType := v.OptionalString("type", "String(UTF8)")
	target := v.OptionalString("target", "")
	scope := v.OptionalString("url", "")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if source == "" && ruleType != "请求头" && ruleType != "响应头" {
		return nil, errors.New("源内容不能为空")
	}

	hash := fmt.Sprintf("%d", time.Now().UnixNano())
	rule := mcp.ConfigReplaceRule{
		Type:  ruleType,
		Src:   source,
		Dest:  target,
		Hash:  hash,
		Scope: scope,
	}
	if c := safeCtx(); c != nil && c.CheckReplace != nil {
		if err := c.CheckReplace(rule); err != nil {
			return nil, err
		}
	}

	c := safeCtx()
	c.TmpLock.Lock()
	rules := c.Config.GetReplaceRules()
	rules = append(rules, rule)
	c.Config.SetReplaceRules(rules)
	_ = c.Config.Save()
	c.TmpLock.Unlock()

	if c.NotifyUI != nil {
		c.NotifyUI("MCP替换规则变更", c.Config.GetReplaceRules())
	}
	return map[string]interface{}{
		"success": true,
		"rule":    rule,
		"index":   len(rules) - 1,
		"message": "替换规则已添加",
	}, nil
}

func toolReplaceRulesRemoveHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	hash := v.OptionalString("hash", "")
	index := v.OptionalInt("index", -1)
	if err := v.Error(); err != nil {
		return nil, err
	}
	if hash == "" && index < 0 {
		return nil, errors.New("请提供 hash 或 index 参数之一")
	}

	c := safeCtx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	rules := c.Config.GetReplaceRules()

	if index >= 0 {
		if index >= len(rules) {
			return nil, fmt.Errorf("索引 %d 超出范围（共 %d 条规则）", index, len(rules))
		}
		newRules := make([]mcp.ConfigReplaceRule, 0, len(rules)-1)
		newRules = append(newRules, rules[:index]...)
		newRules = append(newRules, rules[index+1:]...)
		c.Config.SetReplaceRules(newRules)
		_ = c.Config.Save()
		if c.NotifyUI != nil {
			c.NotifyUI("MCP替换规则变更", newRules)
		}
		return map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("已删除索引 %d 的替换规则", index),
		}, nil
	}

	found := false
	newRules := make([]mcp.ConfigReplaceRule, 0)
	for _, rule := range rules {
		if rule.Hash == hash {
			found = true
			continue
		}
		newRules = append(newRules, rule)
	}
	if !found {
		return nil, fmt.Errorf("未找到Hash为 %s 的规则", hash)
	}
	c.Config.SetReplaceRules(newRules)
	_ = c.Config.Save()
	if c.NotifyUI != nil {
		c.NotifyUI("MCP替换规则变更", newRules)
	}

	return map[string]interface{}{
		"success": true,
		"message": "替换规则已删除",
	}, nil
}

func toolReplaceRulesClearHandler(args map[string]interface{}) (interface{}, error) {
	c := safeCtx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	count := len(c.Config.GetReplaceRules())
	c.Config.SetReplaceRules([]mcp.ConfigReplaceRule{})
	_ = c.Config.Save()

	if c.NotifyUI != nil {
		c.NotifyUI("MCP替换规则变更", []mcp.ConfigReplaceRule{})
	}
	return map[string]interface{}{
		"success": true,
		"cleared": count,
		"message": fmt.Sprintf("已清空 %d 条替换规则", count),
	}, nil
}
