package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	mcpEndpoint string
	httpClient  *http.Client
)

func main() {
	port := flag.Int("port", 29999, "主程序MCP服务端口")
	flag.Parse()

	mcpEndpoint = fmt.Sprintf("http://127.0.0.1:%d/mcp", *port)
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

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
			errResp := errorResponse(nil, -32700, "解析错误", err.Error())
			respBytes, _ := json.Marshal(errResp)
			fmt.Println(string(respBytes))
			continue
		}

		id := request["id"]
		method, _ := request["method"].(string)

		if id == nil || strings.HasPrefix(method, "notifications/") {
			continue
		}

		response := handleRequest(request)
		respBytes, _ := json.Marshal(response)
		fmt.Println(string(respBytes))
	}
}

func handleRequest(request map[string]interface{}) map[string]interface{} {
	method, _ := request["method"].(string)
	id := request["id"]

	switch method {
	case "initialize":
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
		return successResponse(id, map[string]interface{}{})

	case "ping":
		return successResponse(id, map[string]interface{}{})

	case "tools/list":
		return forwardWithRetry(id, request)

	case "tools/call":
		return forwardWithRetry(id, request)

	default:
		return errorResponse(id, -32601, "方法不存在", fmt.Sprintf("未知方法: %s", method))
	}
}

func forwardWithRetry(id interface{}, request map[string]interface{}) map[string]interface{} {
	const maxRetries = 3
	var lastErr string

	for i := 0; i < maxRetries; i++ {
		result, err := forwardToMainProgram(id, request)
		if err == nil {
			return result
		}
		lastErr = err.Error()
		if i < maxRetries-1 {
			time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
		}
	}

	return errorResponse(id, -32603, "连接SunnyNet失败",
		fmt.Sprintf("重试 %d 次后仍无法连接主程序 (%s): %s", maxRetries, mcpEndpoint, lastErr))
}

func forwardToMainProgram(id interface{}, request map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	resp, err := httpClient.Post(mcpEndpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("无法连接到主程序: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("主程序返回状态码 %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取主程序响应失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析主程序响应失败: %v", err)
	}

	result["id"] = id
	return result, nil
}

func successResponse(id interface{}, result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

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
