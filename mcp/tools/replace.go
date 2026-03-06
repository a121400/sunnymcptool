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
		Description: "列出当前所有替换规则",
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
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "源内容（要匹配的内容）",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "替换内容（替换为的内容，响应文件类型时为文件路径）",
				},
			},
			"required": []string{"type", "source", "target"},
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
			},
			"required": []string{"hash"},
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
	rules := ctx().Config.GetReplaceRules()
	if rules == nil {
		rules = []mcp.ConfigReplaceRule{}
	}
	return map[string]interface{}{
		"success": true,
		"rules":   rules,
		"total":   len(rules),
	}, nil
}

func toolReplaceRulesAddHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	ruleType := v.RequireString("type")
	source := v.RequireString("source")
	target := v.OptionalString("target", "")
	if err := v.Error(); err != nil {
		return nil, err
	}

	validTypes := []string{"Base64", "HEX", "String(UTF8)", "String(GBK)", "响应文件"}
	isValidType := false
	for _, t := range validTypes {
		if t == ruleType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return nil, fmt.Errorf("无效的替换类型: %s，支持的类型: %v", ruleType, validTypes)
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

	c := ctx()
	c.TmpLock.Lock()
	rules := c.Config.GetReplaceRules()
	rules = append(rules, rule)
	c.Config.SetReplaceRules(rules)
	_ = c.Config.Save()
	c.TmpLock.Unlock()

	return map[string]interface{}{
		"success": true,
		"rule":    rule,
		"message": "替换规则已添加",
	}, nil
}

func toolReplaceRulesRemoveHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	hash := v.RequireString("hash")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if hash == "" {
		return nil, errors.New("hash不能为空")
	}

	c := ctx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	rules := c.Config.GetReplaceRules()
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

	return map[string]interface{}{
		"success": true,
		"message": "替换规则已删除",
	}, nil
}

func toolReplaceRulesClearHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	c.TmpLock.Lock()
	defer c.TmpLock.Unlock()

	count := len(c.Config.GetReplaceRules())
	c.Config.SetReplaceRules([]mcp.ConfigReplaceRule{})
	_ = c.Config.Save()

	return map[string]interface{}{
		"success": true,
		"cleared": count,
		"message": fmt.Sprintf("已清空 %d 条替换规则", count),
	}, nil
}
