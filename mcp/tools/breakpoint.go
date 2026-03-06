package tools

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/a121400/sunnymcptool/mcp"
)

// BreakpointCondition 断点条件
type BreakpointCondition struct {
	Pattern string         `json:"pattern"`
	Regex   *regexp.Regexp `json:"-"`
}

// BreakpointManager 断点条件管理器
type BreakpointManager struct {
	mu         sync.RWMutex
	conditions []BreakpointCondition
}

// GlobalBreakpointManager 全局断点条件管理器实例
var GlobalBreakpointManager = &BreakpointManager{
	conditions: make([]BreakpointCondition, 0),
}

// Add 添加断点条件
func (m *BreakpointManager) Add(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("URL正则模式无效: %s", err.Error())
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conditions = append(m.conditions, BreakpointCondition{Pattern: pattern, Regex: regex})
	return nil
}

// List 列出所有断点条件
func (m *BreakpointManager) List() []BreakpointCondition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]BreakpointCondition, len(m.conditions))
	copy(result, m.conditions)
	return result
}

// Remove 删除指定索引的断点条件
func (m *BreakpointManager) Remove(index int) (BreakpointCondition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index < 0 || index >= len(m.conditions) {
		return BreakpointCondition{}, fmt.Errorf("索引 %d 超出范围，当前共有 %d 条断点条件（索引范围: 0-%d）",
			index, len(m.conditions), len(m.conditions)-1)
	}
	removed := m.conditions[index]
	m.conditions = append(m.conditions[:index], m.conditions[index+1:]...)
	return removed, nil
}

// Clear 清空所有断点条件
func (m *BreakpointManager) Clear() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := len(m.conditions)
	m.conditions = make([]BreakpointCondition, 0)
	return count
}

// MatchURL 检查URL是否匹配任意断点条件
func (m *BreakpointManager) MatchURL(url string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cond := range m.conditions {
		if cond.Regex.MatchString(url) {
			return true, cond.Pattern
		}
	}
	return false, ""
}

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "breakpoint_add",
		Description: "添加断点条件，指定URL正则模式作为匹配条件，匹配的请求将被自动拦截",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url_pattern": map[string]interface{}{
					"type":        "string",
					"description": "URL正则模式，用于匹配需要拦截的请求URL",
				},
			},
			"required": []string{"url_pattern"},
		},
		Handler: toolBreakpointAddHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "breakpoint_list",
		Description: "列出所有断点条件，返回每条断点的索引和URL正则模式",
		InputSchema: noParamsSchema(),
		Handler:     toolBreakpointListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "breakpoint_remove",
		Description: "删除指定索引的断点条件",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"index": map[string]interface{}{
					"type":        "integer",
					"description": "断点条件索引（从0开始，通过breakpoint_list获取）",
				},
			},
			"required": []string{"index"},
		},
		Handler: toolBreakpointRemoveHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "breakpoint_clear",
		Description: "清空所有断点条件",
		InputSchema: noParamsSchema(),
		Handler:     toolBreakpointClearHandler,
	})
}

func toolBreakpointAddHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	urlPattern := v.RequireString("url_pattern")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if urlPattern == "" {
		return nil, fmt.Errorf("URL正则模式不能为空")
	}
	err := GlobalBreakpointManager.Add(urlPattern)
	if err != nil {
		return nil, err
	}
	conditions := GlobalBreakpointManager.List()
	return map[string]interface{}{
		"success": true,
		"condition": map[string]interface{}{
			"index":       len(conditions) - 1,
			"url_pattern": urlPattern,
		},
		"message": "断点条件已添加",
	}, nil
}

func toolBreakpointListHandler(args map[string]interface{}) (interface{}, error) {
	conditions := GlobalBreakpointManager.List()
	condList := make([]map[string]interface{}, 0, len(conditions))
	for i, cond := range conditions {
		condList = append(condList, map[string]interface{}{
			"index":       i,
			"url_pattern": cond.Pattern,
		})
	}
	return map[string]interface{}{
		"success":    true,
		"conditions": condList,
		"total":      len(condList),
	}, nil
}

func toolBreakpointRemoveHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	index := v.RequireInt("index")
	if err := v.Error(); err != nil {
		return nil, err
	}
	removed, err := GlobalBreakpointManager.Remove(index)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"success": true,
		"removed": map[string]interface{}{"url_pattern": removed.Pattern},
		"message": "断点条件已删除",
	}, nil
}

func toolBreakpointClearHandler(args map[string]interface{}) (interface{}, error) {
	count := GlobalBreakpointManager.Clear()
	return map[string]interface{}{
		"success": true,
		"cleared": count,
		"message": fmt.Sprintf("已清空 %d 条断点条件", count),
	}, nil
}
