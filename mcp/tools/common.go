package tools

import "github.com/a121400/sunnymcptool/mcp"

// ctx 返回全局应用上下文
func ctx() *mcp.AppContext {
	return mcp.GlobalAppContext
}

// noParamsSchema 返回无参数的输入Schema
func noParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}
