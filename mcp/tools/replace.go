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
		Description: "添加新的替换规则，支持类型：Base64、HEX、String(UTF8)、String(GBK)、响应文件",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type":        "string",
					"description": "替换类型：Base64、HEX、String(UTF8)、String(GBK)、响应文件",
					"enum":        []string{"Base64", "HEX", "String(UTF8)", "String(GBK)", "响应文件"},
					"default":     "String(UTF8)",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "源内容（要匹配的内容）",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "替换内容（替换为的内容，响应文件类型时为文件路径）",
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
					"description": "规则索引（从0开始，通过breakpoint_list获取）",
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
	if err := v.Error(); err != nil {
		return nil, err
	}

	validTypes := map[string]bool{
		"Base64": true, "HEX": true, "String(UTF8)": true, "String(GBK)": true, "响应文件": true,
	}
	if !validTypes[ruleType] {
		return nil, fmt.Errorf("无效的替换类型: %s，支持: Base64, HEX, String(UTF8), String(GBK), 响应文件", ruleType)
	}
	if source == "" {
		return nil, errors.New("源内容不能为空")
	}

	hash := fmt.Sprintf("%d", time.Now().UnixNano())
	rule := mcp.ConfigReplaceRule{
		Type: ruleType,
		Src:  source,
		Dest: target,
		Hash: hash,
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
