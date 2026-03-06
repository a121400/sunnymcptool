package mcp

import (
	"strings"
	"testing"
)

// ============ NewParamValidator 测试 ============

func TestNewParamValidator_NilArgs(t *testing.T) {
	v := NewParamValidator(nil)
	if v == nil {
		t.Fatal("NewParamValidator(nil) 不应返回 nil")
	}
	if v.args == nil {
		t.Fatal("args 不应为 nil")
	}
	if v.Error() != nil {
		t.Fatal("空验证器不应有错误")
	}
}

func TestNewParamValidator_EmptyArgs(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	if v == nil {
		t.Fatal("NewParamValidator 不应返回 nil")
	}
	if v.Error() != nil {
		t.Fatal("空验证器不应有错误")
	}
}

// ============ RequireInt 测试 ============

func TestRequireInt_Float64(t *testing.T) {
	args := map[string]interface{}{"port": float64(8080)}
	v := NewParamValidator(args)
	result := v.RequireInt("port")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != 8080 {
		t.Fatalf("期望 8080, 实际 %d", result)
	}
}

func TestRequireInt_NativeInt(t *testing.T) {
	args := map[string]interface{}{"count": 42}
	v := NewParamValidator(args)
	result := v.RequireInt("count")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != 42 {
		t.Fatalf("期望 42, 实际 %d", result)
	}
}

func TestRequireInt_Missing(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	result := v.RequireInt("theology")
	if result != 0 {
		t.Fatalf("缺失参数应返回零值, 实际 %d", result)
	}
	err := v.Error()
	if err == nil {
		t.Fatal("缺失必需参数应返回错误")
	}
	if !strings.Contains(err.Error(), "theology") {
		t.Fatalf("错误信息应包含参数名 'theology': %v", err)
	}
}

func TestRequireInt_WrongType(t *testing.T) {
	args := map[string]interface{}{"port": "not_a_number"}
	v := NewParamValidator(args)
	result := v.RequireInt("port")
	if result != 0 {
		t.Fatalf("类型错误应返回零值, 实际 %d", result)
	}
	err := v.Error()
	if err == nil {
		t.Fatal("类型不匹配应返回错误")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "port") {
		t.Fatalf("错误信息应包含参数名 'port': %s", errMsg)
	}
	if !strings.Contains(errMsg, "int") {
		t.Fatalf("错误信息应包含期望类型 'int': %s", errMsg)
	}
}

func TestRequireInt_Zero(t *testing.T) {
	args := map[string]interface{}{"offset": float64(0)}
	v := NewParamValidator(args)
	result := v.RequireInt("offset")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != 0 {
		t.Fatalf("期望 0, 实际 %d", result)
	}
}

func TestRequireInt_Negative(t *testing.T) {
	args := map[string]interface{}{"value": float64(-100)}
	v := NewParamValidator(args)
	result := v.RequireInt("value")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != -100 {
		t.Fatalf("期望 -100, 实际 %d", result)
	}
}

// ============ RequireString 测试 ============

func TestRequireString_Normal(t *testing.T) {
	args := map[string]interface{}{"key": "Content-Type"}
	v := NewParamValidator(args)
	result := v.RequireString("key")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != "Content-Type" {
		t.Fatalf("期望 'Content-Type', 实际 '%s'", result)
	}
}

func TestRequireString_Empty(t *testing.T) {
	args := map[string]interface{}{"body": ""}
	v := NewParamValidator(args)
	result := v.RequireString("body")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != "" {
		t.Fatalf("期望空字符串, 实际 '%s'", result)
	}
}

func TestRequireString_Missing(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	result := v.RequireString("name")
	if result != "" {
		t.Fatalf("缺失参数应返回空字符串, 实际 '%s'", result)
	}
	err := v.Error()
	if err == nil {
		t.Fatal("缺失必需参数应返回错误")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("错误信息应包含参数名 'name': %v", err)
	}
}

func TestRequireString_WrongType(t *testing.T) {
	args := map[string]interface{}{"key": float64(123)}
	v := NewParamValidator(args)
	result := v.RequireString("key")
	if result != "" {
		t.Fatalf("类型错误应返回空字符串, 实际 '%s'", result)
	}
	err := v.Error()
	if err == nil {
		t.Fatal("类型不匹配应返回错误")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "key") {
		t.Fatalf("错误信息应包含参数名 'key': %s", errMsg)
	}
}

// ============ OptionalInt 测试 ============

func TestOptionalInt_Present(t *testing.T) {
	args := map[string]interface{}{"limit": float64(200)}
	v := NewParamValidator(args)
	result := v.OptionalInt("limit", 100)
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != 200 {
		t.Fatalf("期望 200, 实际 %d", result)
	}
}

func TestOptionalInt_Missing(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	result := v.OptionalInt("limit", 50)
	if err := v.Error(); err != nil {
		t.Fatalf("可选参数缺失不应有错误: %v", err)
	}
	if result != 50 {
		t.Fatalf("期望默认值 50, 实际 %d", result)
	}
}

func TestOptionalInt_WrongType(t *testing.T) {
	args := map[string]interface{}{"limit": "abc"}
	v := NewParamValidator(args)
	result := v.OptionalInt("limit", 100)
	if result != 100 {
		t.Fatalf("类型错误应返回默认值 100, 实际 %d", result)
	}
	if v.Error() == nil {
		t.Fatal("类型不匹配应返回错误")
	}
}

func TestOptionalInt_NativeInt(t *testing.T) {
	args := map[string]interface{}{"offset": 25}
	v := NewParamValidator(args)
	result := v.OptionalInt("offset", 0)
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != 25 {
		t.Fatalf("期望 25, 实际 %d", result)
	}
}

// ============ OptionalString 测试 ============

func TestOptionalString_Present(t *testing.T) {
	args := map[string]interface{}{"method": "GET"}
	v := NewParamValidator(args)
	result := v.OptionalString("method", "POST")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if result != "GET" {
		t.Fatalf("期望 'GET', 实际 '%s'", result)
	}
}

func TestOptionalString_Missing(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	result := v.OptionalString("method", "GET")
	if err := v.Error(); err != nil {
		t.Fatalf("可选参数缺失不应有错误: %v", err)
	}
	if result != "GET" {
		t.Fatalf("期望默认值 'GET', 实际 '%s'", result)
	}
}

func TestOptionalString_WrongType(t *testing.T) {
	args := map[string]interface{}{"method": float64(123)}
	v := NewParamValidator(args)
	result := v.OptionalString("method", "GET")
	if result != "GET" {
		t.Fatalf("类型错误应返回默认值 'GET', 实际 '%s'", result)
	}
	if v.Error() == nil {
		t.Fatal("类型不匹配应返回错误")
	}
}

// ============ Error 多错误收集测试 ============

func TestError_MultipleErrors(t *testing.T) {
	v := NewParamValidator(map[string]interface{}{})
	v.RequireInt("id")
	v.RequireString("name")
	err := v.Error()
	if err == nil {
		t.Fatal("应有多个错误")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "id") {
		t.Fatalf("错误信息应包含 'id': %s", errMsg)
	}
	if !strings.Contains(errMsg, "name") {
		t.Fatalf("错误信息应包含 'name': %s", errMsg)
	}
}

func TestError_NoErrors(t *testing.T) {
	args := map[string]interface{}{"id": float64(1), "name": "test"}
	v := NewParamValidator(args)
	v.RequireInt("id")
	v.RequireString("name")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
}

// ============ 综合使用场景测试 ============

func TestValidator_TypicalUsage(t *testing.T) {
	args := map[string]interface{}{
		"theology": float64(12345),
		"key":      "Content-Type",
		"value":    "application/json",
	}
	v := NewParamValidator(args)
	theology := v.RequireInt("theology")
	key := v.RequireString("key")
	value := v.RequireString("value")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if theology != 12345 || key != "Content-Type" || value != "application/json" {
		t.Fatal("参数值不匹配")
	}
}

func TestValidator_MixedRequiredAndOptional(t *testing.T) {
	args := map[string]interface{}{"theology": float64(100)}
	v := NewParamValidator(args)
	theology := v.RequireInt("theology")
	limit := v.OptionalInt("limit", 50)
	method := v.OptionalString("method", "GET")
	if err := v.Error(); err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if theology != 100 || limit != 50 || method != "GET" {
		t.Fatal("参数值不匹配")
	}
}
