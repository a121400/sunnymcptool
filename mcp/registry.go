package mcp

import (
	"fmt"
	"sync"
)

// MCPTool MCP工具定义结构
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolHandlerFunc 工具处理函数签名
type ToolHandlerFunc func(args map[string]interface{}) (interface{}, error)

// ToolDefinition 工具完整定义
type ToolDefinition struct {
	Name        string                 // 工具名称
	Description string                 // 工具描述
	InputSchema map[string]interface{} // 输入参数Schema
	Handler     ToolHandlerFunc        // 处理函数
}

// ToolRegistry 工具注册中心
// 负责工具的注册、查找和调用分发，使用读写锁保证并发安全
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDefinition
	order []string // 保持注册顺序
}

// NewToolRegistry 创建新的工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDefinition),
		order: make([]string, 0),
	}
}

// GlobalRegistry 全局工具注册中心实例
var GlobalRegistry = NewToolRegistry()

// Register 注册工具到注册中心
// 如果同名工具已存在，则覆盖旧注册（保持原有顺序位置）
func (r *ToolRegistry) Register(def ToolDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[def.Name]; !exists {
		r.order = append(r.order, def.Name)
	}
	r.tools[def.Name] = &ToolDefinition{
		Name:        def.Name,
		Description: def.Description,
		InputSchema: def.InputSchema,
		Handler:     def.Handler,
	}
}

// GetToolsList 获取所有已注册工具的MCP定义列表
// 返回的工具列表按注册顺序排列
func (r *ToolRegistry) GetToolsList() []MCPTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]MCPTool, 0, len(r.order))
	for _, name := range r.order {
		if def, exists := r.tools[name]; exists {
			result = append(result, MCPTool{
				Name:        def.Name,
				Description: def.Description,
				InputSchema: def.InputSchema,
			})
		}
	}
	return result
}

// Call 调用指定名称的工具
// 如果工具未注册，返回包含工具名称的错误信息
func (r *ToolRegistry) Call(name string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	def, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("工具 '%s' 未注册", name)
	}

	return def.Handler(args)
}
