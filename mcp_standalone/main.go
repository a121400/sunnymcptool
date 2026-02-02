package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// 独立的MCP服务器 - 通过stdio与Cursor通信，快速启动

func main() {
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

func handleNotification(request map[string]interface{}) {
	// 通知消息不需要响应，静默处理
}

func handleRequest(request map[string]interface{}) map[string]interface{} {
	method, _ := request["method"].(string)
	id := request["id"]
	params, _ := request["params"].(map[string]interface{})

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
		return successResponse(id, map[string]interface{}{
			"tools": getToolsList(),
		})

	case "tools/call":
		return handleToolCall(id, params)

	default:
		return errorResponse(id, -32601, "Method not found", method)
	}
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

func toolResult(text string, isError bool) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"isError": isError,
	}
}

func getToolsList() []map[string]interface{} {
	return []map[string]interface{}{
		tool("proxy_start", "启动SunnyNet代理服务", nil),
		tool("proxy_stop", "停止SunnyNet代理服务", nil),
		tool("proxy_set_port", "设置代理端口号", map[string]interface{}{
			"port": prop("integer", "代理端口号 (1-65535)"),
		}, "port"),
		tool("proxy_get_status", "获取代理服务状态", nil),
		tool("request_list", "获取已捕获的HTTP请求列表", map[string]interface{}{
			"limit":  prop("integer", "返回的最大数量，默认100"),
			"offset": prop("integer", "偏移量，用于分页"),
		}),
		tool("request_get", "获取指定请求的详细信息", map[string]interface{}{
			"theology": prop("integer", "请求的唯一ID"),
		}, "theology"),
		tool("config_get", "获取SunnyNet当前配置信息", nil),
		tool("cert_install", "安装CA证书到系统信任列表", nil),
		tool("cert_export", "导出CA证书到指定路径", map[string]interface{}{
			"path": prop("string", "证书导出路径"),
		}, "path"),
		tool("process_list", "获取当前系统运行的进程列表", nil),
		tool("process_add_name", "添加要拦截的进程名", map[string]interface{}{
			"name": prop("string", "进程名称"),
		}, "name"),
		tool("process_remove_name", "移除已添加的拦截进程名", map[string]interface{}{
			"name": prop("string", "进程名称"),
		}, "name"),
		tool("request_modify_header", "修改指定请求的请求头", map[string]interface{}{
			"theology": prop("integer", "请求ID"),
			"key":      prop("string", "请求头名称"),
			"value":    prop("string", "请求头值"),
		}, "theology", "key", "value"),
		tool("request_modify_body", "修改指定请求的请求体", map[string]interface{}{
			"theology": prop("integer", "请求ID"),
			"body":     prop("string", "新的请求体"),
		}, "theology", "body"),
		tool("response_modify_header", "修改指定请求的响应头", map[string]interface{}{
			"theology": prop("integer", "请求ID"),
			"key":      prop("string", "响应头名称"),
			"value":    prop("string", "响应头值"),
		}, "theology", "key", "value"),
		tool("response_modify_body", "修改指定请求的响应体", map[string]interface{}{
			"theology": prop("integer", "请求ID"),
			"body":     prop("string", "新的响应体"),
		}, "theology", "body"),
		tool("request_block", "阻断/拦截指定的请求", map[string]interface{}{
			"theology": prop("integer", "请求ID"),
		}, "theology"),
		tool("request_release_all", "放行所有被拦截的请求", nil),
		tool("decrypt_packet", "解密单个数据包", map[string]interface{}{
			"data": prop("string", "数据包的十六进制字符串"),
		}, "data"),
		tool("parse_protobuf", "解析Protobuf数据", map[string]interface{}{
			"data": prop("string", "Protobuf数据的十六进制字符串"),
		}, "data"),
		tool("crypto_config_get", "获取当前加密配置", nil),
		tool("crypto_config_set", "设置加密配置", map[string]interface{}{
			"name":        prop("string", "配置名称"),
			"aes_key":     prop("string", "AES密钥"),
			"aes_iv":      prop("string", "AES IV"),
			"header_size": prop("integer", "头部大小"),
		}, "name", "aes_key", "aes_iv"),
		tool("crypto_config_list", "列出所有加密配置", nil),
		tool("decrypt_tcp_flow", "解密TCP连接数据流", map[string]interface{}{
			"theology": prop("integer", "TCP连接ID"),
		}, "theology"),
		// 替换规则类
		tool("replace_rules_list", "列出当前所有替换规则", nil),
		tool("replace_rules_add", "添加新的替换规则", map[string]interface{}{
			"type":   prop("string", "替换类型：Base64、HEX、String(UTF8)、String(GBK)、响应文件"),
			"source": prop("string", "源内容（要匹配的内容）"),
			"target": prop("string", "替换内容"),
		}, "type", "source", "target"),
		tool("replace_rules_remove", "删除指定的替换规则", map[string]interface{}{
			"hash": prop("string", "规则的唯一标识Hash"),
		}, "hash"),
		tool("replace_rules_clear", "清空所有替换规则", nil),
	}
}

func tool(name, desc string, props map[string]interface{}, required ...string) map[string]interface{} {
	if props == nil {
		props = map[string]interface{}{}
	}
	if required == nil {
		required = []string{}
	}
	return map[string]interface{}{
		"name":        name,
		"description": desc,
		"inputSchema": map[string]interface{}{
			"type":       "object",
			"properties": props,
			"required":   required,
		},
	}
}

func prop(typ, desc string) map[string]interface{} {
	return map[string]interface{}{
		"type":        typ,
		"description": desc,
	}
}

func handleToolCall(id interface{}, params map[string]interface{}) map[string]interface{} {
	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = map[string]interface{}{}
	}

	result := callSunnyNetAPI(name, args)
	return successResponse(id, toolResult(result, false))
}

func callSunnyNetAPI(toolName string, args map[string]interface{}) string {
	body := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	jsonBody, _ := json.Marshal(body)
	resp, err := http.Post("http://127.0.0.1:29999/mcp", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Sprintf(`{"error": "连接SunnyNet失败: %v"}`, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	if res, ok := result["result"].(map[string]interface{}); ok {
		if content, ok := res["content"].([]interface{}); ok && len(content) > 0 {
			if first, ok := content[0].(map[string]interface{}); ok {
				if text, ok := first["text"].(string); ok {
					return text
				}
			}
		}
	}

	return string(respBody)
}
