package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// 独立的MCP服务器 - 通过stdio与AI工具通信
// 工具列表和工具调用全部通过HTTP转发到主程序，不再本地硬编码工具定义

// 主程序MCP端点地址
var mcpEndpoint string

func main() {
	// 命令行参数配置主程序端口
	port := flag.Int("port", 29999, "主程序MCP服务端口")
	flag.Parse()

	mcpEndpoint = fmt.Sprintf("http://127.0.0.1:%d/mcp", *port)

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			continue
		}

		// 检查是否是通知消息（没有id的请求不需要响应）
		id := request["id"]
		method, _ := request["method"].(string)

		// 通知消息不需要响应
		if id == nil || strings.HasPrefix(method, "notifications/") {
			handleNotification(request)
			continue
		}

		response := handleRequest(request)
		respBytes, _ := json.Marshal(response)
		fmt.Println(string(respBytes))
	}
}

// handleNotification 处理通知消息，不返回响应
func handleNotification(request map[string]interface{}) {
	// 通知消息不需要响应，静默处理
}

// handleRequest 处理JSON-RPC请求并返回响应
func handleRequest(request map[string]interface{}) map[string]interface{} {
	method, _ := request["method"].(string)
	id := request["id"]

	switch method {
	case "initialize":
		// 本地处理初始化请求
		return successResponse(id, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{"listChanged": true},
			},
			"serverInfo": map[string]interface{}{
				"name":    "SunnyNet-MCP",
				"version": "1.0.0",
			},
		})

	case "initialized", "notifications/initialized":
		// 本地处理初始化完成确认
		return successResponse(id, map[string]interface{}{})

	case "ping":
		// 本地处理ping
		return successResponse(id, map[string]interface{}{})

	case "tools/list":
		// 转发到主程序获取工具列表
		return forwardToMainProgram(id, request)

	case "tools/call":
		// 转发到主程序执行工具调用
		return forwardToMainProgram(id, request)

	default:
		return errorResponse(id, -32601, "方法不存在", fmt.Sprintf("未知方法: %s", method))
	}
}

// forwardToMainProgram 将JSON-RPC请求转发到主程序的MCP端点
// 主程序返回的响应直接透传回调用方
func forwardToMainProgram(id interface{}, request map[string]interface{}) map[string]interface{} {
	// 构建转发请求体
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return errorResponse(id, -32603, "内部错误", fmt.Sprintf("序列化请求失败: %v", err))
	}

	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(mcpEndpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		// 连接失败时返回包含错误详情的响应
		return errorResponse(id, -32603, "连接SunnyNet失败",
			fmt.Sprintf("无法连接到主程序 (%s): %v", mcpEndpoint, err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorResponse(id, -32603, "读取响应失败",
			fmt.Sprintf("读取主程序响应失败: %v", err))
	}

	// 解析主程序返回的JSON-RPC响应并透传
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return errorResponse(id, -32603, "解析响应失败",
			fmt.Sprintf("解析主程序响应失败: %v", err))
	}

	// 确保响应中的id与请求一致
	result["id"] = id

	return result
}

// successResponse 构建JSON-RPC成功响应
func successResponse(id interface{}, result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

// errorResponse 构建JSON-RPC错误响应
func errorResponse(id interface{}, code int, message string, data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"data":    data,
		},
	}
}
