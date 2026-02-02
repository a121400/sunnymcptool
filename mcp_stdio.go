// +build ignore

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// stdio模式的MCP服务器
// 通过标准输入/输出与Cursor通信

func main() {
	reader := bufio.NewReader(os.Stdin)
	
	for {
		// 读取一行JSON-RPC请求
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 解析请求
		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			continue
		}
		
		// 处理请求
		response := handleStdioRequest(request)
		
		// 输出响应
		respBytes, _ := json.Marshal(response)
		fmt.Println(string(respBytes))
	}
}

func handleStdioRequest(request map[string]interface{}) map[string]interface{} {
	method, _ := request["method"].(string)
	id := request["id"]
	
	switch method {
	case "initialize":
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": true,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "SunnyNet-MCP",
					"version": "1.0.0",
				},
			},
		}
	
	case "initialized":
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  map[string]interface{}{},
		}
	
	case "tools/list":
		// 通过HTTP调用主程序获取工具列表
		tools := getToolsFromMain()
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"tools": tools,
			},
		}
	
	case "tools/call":
		params, _ := request["params"].(map[string]interface{})
		result := callToolOnMain(params)
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}
	
	case "ping":
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  map[string]interface{}{},
		}
	
	default:
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
	}
}

func getToolsFromMain() []interface{} {
	resp, err := http.Post("http://127.0.0.1:29999/mcp", "application/json", 
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	if err != nil {
		return []interface{}{}
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if res, ok := result["result"].(map[string]interface{}); ok {
		if tools, ok := res["tools"].([]interface{}); ok {
			return tools
		}
	}
	return []interface{}{}
}

func callToolOnMain(params map[string]interface{}) map[string]interface{} {
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  params,
	})
	
	resp, err := http.Post("http://127.0.0.1:29999/mcp", "application/json", 
		strings.NewReader(string(body)))
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
			},
			"isError": true,
		}
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if res, ok := result["result"].(map[string]interface{}); ok {
		return res
	}
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": "Unknown error"},
		},
		"isError": true,
	}
}
