package mcp

import (
	"fmt"
	"strings"
)

// ParamValidator 参数验证器
// 从 map[string]interface{} 中提取和验证工具参数，
// 收集所有错误后一次性返回，错误信息包含参数名和期望类型
type ParamValidator struct {
	args   map[string]interface{}
	errors []string
}

// NewParamValidator 创建参数验证器
func NewParamValidator(args map[string]interface{}) *ParamValidator {
	if args == nil {
		args = make(map[string]interface{})
	}
	return &ParamValidator{
		args:   args,
		errors: make([]string, 0),
	}
}

// RequireInt 提取必需的整数参数
func (v *ParamValidator) RequireInt(name string) int {
	val, exists := v.args[name]
	if !exists {
		v.errors = append(v.errors, fmt.Sprintf("缺少必需参数 '%s' (期望类型: int)", name))
		return 0
	}
	switch n := val.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 类型错误: 期望 int, 实际 %T", name, val))
		return 0
	}
}

// RequireString 提取必需的字符串参数
func (v *ParamValidator) RequireString(name string) string {
	val, exists := v.args[name]
	if !exists {
		v.errors = append(v.errors, fmt.Sprintf("缺少必需参数 '%s' (期望类型: string)", name))
		return ""
	}
	s, ok := val.(string)
	if !ok {
		v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 类型错误: 期望 string, 实际 %T", name, val))
		return ""
	}
	return s
}

// OptionalInt 提取可选整数参数，缺失时返回默认值
func (v *ParamValidator) OptionalInt(name string, defaultVal int) int {
	val, exists := v.args[name]
	if !exists {
		return defaultVal
	}
	switch n := val.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 类型错误: 期望 int, 实际 %T", name, val))
		return defaultVal
	}
}

// OptionalString 提取可选字符串参数，缺失时返回默认值
func (v *ParamValidator) OptionalString(name string, defaultVal string) string {
	val, exists := v.args[name]
	if !exists {
		return defaultVal
	}
	s, ok := val.(string)
	if !ok {
		v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 类型错误: 期望 string, 实际 %T", name, val))
		return defaultVal
	}
	return s
}

// RequireIntArray 提取必需的整数数组参数
func (v *ParamValidator) RequireIntArray(name string) []int {
	val, exists := v.args[name]
	if !exists {
		v.errors = append(v.errors, fmt.Sprintf("缺少必需参数 '%s' (期望类型: []int)", name))
		return nil
	}
	arr, ok := val.([]interface{})
	if !ok {
		v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 类型错误: 期望 []int, 实际 %T", name, val))
		return nil
	}
	result := make([]int, 0, len(arr))
	for _, item := range arr {
		switch n := item.(type) {
		case float64:
			result = append(result, int(n))
		case int:
			result = append(result, n)
		default:
			v.errors = append(v.errors, fmt.Sprintf("参数 '%s' 数组元素类型错误: 期望 int, 实际 %T", name, item))
			return nil
		}
	}
	return result
}

// Error 返回验证错误，无错误返回 nil
func (v *ParamValidator) Error() error {
	if len(v.errors) == 0 {
		return nil
	}
	return fmt.Errorf("参数验证失败: %s", strings.Join(v.errors, "; "))
}
