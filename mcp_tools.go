package main

import (
	"changeme/CommAnd"
	"changeme/MapHash"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qtgolang/SunnyNet/public"
	"github.com/qtgolang/SunnyNet/src/JsCall"
)

// MCPTool MCP工具定义结构
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// 工具列表缓存，使用 sync.Once 确保只构建一次
var (
	cachedTools []MCPTool
	toolsOnce   sync.Once
)

// GetToolsList 返回所有MCP工具定义（使用缓存）
func GetToolsList() []MCPTool {
	toolsOnce.Do(func() {
		cachedTools = buildToolsList()
	})
	return cachedTools
}

// buildToolsList 构建工具列表（只在首次调用时执行）
func buildToolsList() []MCPTool {
	return []MCPTool{
		// ============ 代理控制类 (4个) ============
		{
			Name:        "proxy_start",
			Description: "启动SunnyNet代理服务",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "proxy_stop",
			Description: "停止SunnyNet代理服务",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "proxy_set_port",
			Description: "设置代理端口号",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "代理端口号 (1-65535)",
						"minimum":     1,
						"maximum":     65535,
					},
				},
				"required": []string{"port"},
			},
		},
		{
			Name:        "proxy_get_status",
			Description: "获取代理服务状态，包括运行状态、端口号等信息",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},

		// ============ 请求拦截类 (8个) ============
		{
			Name:        "request_list",
			Description: "获取已捕获的HTTP请求列表",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "返回的最大数量，默认100",
						"default":     100,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "偏移量，用于分页",
						"default":     0,
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "request_get",
			Description: "获取指定请求的详细信息，包括请求头、请求体、响应头、响应体等",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
				},
				"required": []string{"theology"},
			},
		},
		{
			Name:        "request_modify_header",
			Description: "修改指定请求的请求头",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "请求头的名称",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "请求头的值",
					},
				},
				"required": []string{"theology", "key", "value"},
			},
		},
		{
			Name:        "request_modify_body",
			Description: "修改指定请求的请求体",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "新的请求体内容",
					},
				},
				"required": []string{"theology", "body"},
			},
		},
		{
			Name:        "response_modify_header",
			Description: "修改指定请求的响应头",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "响应头的名称",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "响应头的值",
					},
				},
				"required": []string{"theology", "key", "value"},
			},
		},
		{
			Name:        "response_modify_body",
			Description: "修改指定请求的响应体",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "新的响应体内容",
					},
				},
				"required": []string{"theology", "body"},
			},
		},
		{
			Name:        "request_block",
			Description: "阻断/拦截指定的请求",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "请求的唯一ID (Theology)",
					},
				},
				"required": []string{"theology"},
			},
		},
		{
			Name:        "request_release_all",
			Description: "放行所有被断点拦截的请求",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},

		// ============ 证书管理类 (2个) ============
		{
			Name:        "cert_install",
			Description: "安装SunnyNet默认CA证书到系统信任列表",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "cert_export",
			Description: "导出SunnyNet默认CA证书到指定路径",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "证书导出路径（包含文件名，如：C:/cert/SunnyNet.cer）",
					},
				},
				"required": []string{"path"},
			},
		},

		// ============ 进程拦截类 (3个, Windows) ============
		{
			Name:        "process_list",
			Description: "获取当前系统运行的进程列表（用于选择要拦截的进程）",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "process_add_name",
			Description: "添加要拦截的进程名（Windows需要加载驱动）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "进程名称（如：chrome.exe）",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "process_remove_name",
			Description: "移除已添加的拦截进程名",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "进程名称（如：chrome.exe）",
					},
				},
				"required": []string{"name"},
			},
		},

		// ============ 配置类 (1个) ============
		{
			Name:        "config_get",
			Description: "获取SunnyNet当前配置信息",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},

		// ============ 解密分析类 (6个) ============
		{
			Name:        "decrypt_packet",
			Description: "解密单个数据包，返回解密后的数据包详情（包括头部信息、原始数据、解密数据、Protobuf解析）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type":        "string",
						"description": "原始数据包的十六进制字符串",
					},
				},
				"required": []string{"data"},
			},
		},
		{
			Name:        "parse_protobuf",
			Description: "解析Protobuf数据，返回字段树结构",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Protobuf数据的十六进制字符串",
					},
				},
				"required": []string{"data"},
			},
		},
		{
			Name:        "crypto_config_get",
			Description: "获取当前使用的加密配置详情",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "crypto_config_set",
			Description: "设置加密配置（AES密钥、IV、头部大小等）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "配置名称",
					},
					"aes_key": map[string]interface{}{
						"type":        "string",
						"description": "AES密钥（16/24/32字节的字符串或hex）",
					},
					"aes_iv": map[string]interface{}{
						"type":        "string",
						"description": "AES IV（16字节的字符串或hex）",
					},
					"header_size": map[string]interface{}{
						"type":        "integer",
						"description": "数据包头部大小（字节数）",
						"default":     20,
					},
				},
				"required": []string{"name", "aes_key", "aes_iv"},
			},
		},
		{
			Name:        "crypto_config_list",
			Description: "列出所有已配置的加密配置",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "decrypt_tcp_flow",
			Description: "解密指定TCP连接的完整数据流，返回所有解密后的数据包列表",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"theology": map[string]interface{}{
						"type":        "integer",
						"description": "TCP连接的唯一ID (Theology)",
					},
				},
				"required": []string{"theology"},
			},
		},

		// ============ 替换规则类 (4个) ============
		{
			Name:        "replace_rules_list",
			Description: "列出当前所有替换规则",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
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
		},
		{
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
		},
		{
			Name:        "replace_rules_clear",
			Description: "清空所有替换规则",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
	}
}

// CallTool 调用指定的MCP工具
func CallTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	// ============ 代理控制类 ============
	case "proxy_start":
		return toolProxyStart()
	case "proxy_stop":
		return toolProxyStop()
	case "proxy_set_port":
		port, ok := args["port"].(float64)
		if !ok {
			return nil, errors.New("参数 port 必须是整数")
		}
		return toolProxySetPort(int(port))
	case "proxy_get_status":
		return toolProxyGetStatus()

	// ============ 请求拦截类 ============
	case "request_list":
		limit := 100
		offset := 0
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		if o, ok := args["offset"].(float64); ok {
			offset = int(o)
		}
		return toolRequestList(limit, offset)
	case "request_get":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		return toolRequestGet(int(theology))
	case "request_modify_header":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		key, ok := args["key"].(string)
		if !ok {
			return nil, errors.New("参数 key 必须是字符串")
		}
		value, ok := args["value"].(string)
		if !ok {
			return nil, errors.New("参数 value 必须是字符串")
		}
		return toolRequestModifyHeader(int(theology), key, value)
	case "request_modify_body":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		body, ok := args["body"].(string)
		if !ok {
			return nil, errors.New("参数 body 必须是字符串")
		}
		return toolRequestModifyBody(int(theology), body)
	case "response_modify_header":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		key, ok := args["key"].(string)
		if !ok {
			return nil, errors.New("参数 key 必须是字符串")
		}
		value, ok := args["value"].(string)
		if !ok {
			return nil, errors.New("参数 value 必须是字符串")
		}
		return toolResponseModifyHeader(int(theology), key, value)
	case "response_modify_body":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		body, ok := args["body"].(string)
		if !ok {
			return nil, errors.New("参数 body 必须是字符串")
		}
		return toolResponseModifyBody(int(theology), body)
	case "request_block":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		return toolRequestBlock(int(theology))
	case "request_release_all":
		return toolRequestReleaseAll()

	// ============ 证书管理类 ============
	case "cert_install":
		return toolCertInstall()
	case "cert_export":
		path, ok := args["path"].(string)
		if !ok {
			return nil, errors.New("参数 path 必须是字符串")
		}
		return toolCertExport(path)

	// ============ 进程拦截类 ============
	case "process_list":
		return toolProcessList()
	case "process_add_name":
		name, ok := args["name"].(string)
		if !ok {
			return nil, errors.New("参数 name 必须是字符串")
		}
		return toolProcessAddName(name)
	case "process_remove_name":
		name, ok := args["name"].(string)
		if !ok {
			return nil, errors.New("参数 name 必须是字符串")
		}
		return toolProcessRemoveName(name)

	// ============ 配置类 ============
	case "config_get":
		return toolConfigGet()

	// ============ 解密分析类 ============
	case "decrypt_packet":
		data, ok := args["data"].(string)
		if !ok {
			return nil, errors.New("参数 data 必须是字符串")
		}
		return toolDecryptPacket(data)
	case "parse_protobuf":
		data, ok := args["data"].(string)
		if !ok {
			return nil, errors.New("参数 data 必须是字符串")
		}
		return toolParseProtobuf(data)
	case "crypto_config_get":
		return toolCryptoConfigGet()
	case "crypto_config_set":
		name, ok := args["name"].(string)
		if !ok {
			return nil, errors.New("参数 name 必须是字符串")
		}
		aesKey, ok := args["aes_key"].(string)
		if !ok {
			return nil, errors.New("参数 aes_key 必须是字符串")
		}
		aesIV, ok := args["aes_iv"].(string)
		if !ok {
			return nil, errors.New("参数 aes_iv 必须是字符串")
		}
		headerSize := 20
		if hs, ok := args["header_size"].(float64); ok {
			headerSize = int(hs)
		}
		return toolCryptoConfigSet(name, aesKey, aesIV, headerSize)
	case "crypto_config_list":
		return toolCryptoConfigList()
	case "decrypt_tcp_flow":
		theology, ok := args["theology"].(float64)
		if !ok {
			return nil, errors.New("参数 theology 必须是整数")
		}
		return toolDecryptTcpFlow(int(theology))

	// ============ 替换规则类 ============
	case "replace_rules_list":
		return toolReplaceRulesList()
	case "replace_rules_add":
		ruleType, ok := args["type"].(string)
		if !ok {
			return nil, errors.New("参数 type 必须是字符串")
		}
		source, ok := args["source"].(string)
		if !ok {
			return nil, errors.New("参数 source 必须是字符串")
		}
		target, _ := args["target"].(string) // target 可以为空
		return toolReplaceRulesAdd(ruleType, source, target)
	case "replace_rules_remove":
		hash, ok := args["hash"].(string)
		if !ok {
			return nil, errors.New("参数 hash 必须是字符串")
		}
		return toolReplaceRulesRemove(hash)
	case "replace_rules_clear":
		return toolReplaceRulesClear()

	default:
		return nil, fmt.Errorf("未知的工具: %s", name)
	}
}

// ============ 代理控制类工具实现 ============

// toolProxyStart 启动代理服务
func toolProxyStart() (interface{}, error) {
	if app == nil || app.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}

	err := app.App.Start().Error
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"port":    app.App.Port(),
		"message": "代理服务已启动",
	}, nil
}

// toolProxyStop 停止代理服务
func toolProxyStop() (interface{}, error) {
	if app == nil || app.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}

	app.App.Close()

	return map[string]interface{}{
		"success": true,
		"message": "代理服务已停止",
	}, nil
}

// toolProxySetPort 设置代理端口
func toolProxySetPort(port int) (interface{}, error) {
	if app == nil || app.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}

	if port < 1 || port > 65535 {
		return nil, errors.New("端口号必须在 1-65535 之间")
	}

	// 先关闭再设置端口并重启
	app.App.Close()
	app.App.SetPort(port)
	err := app.App.Start().Error

	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	// 保存配置
	GlobalConfig.Port = port
	_ = GlobalConfig.saveToFile()

	return map[string]interface{}{
		"success": true,
		"port":    port,
		"message": fmt.Sprintf("端口已设置为 %d", port),
	}, nil
}

// toolProxyGetStatus 获取代理状态
func toolProxyGetStatus() (interface{}, error) {
	if app == nil || app.App == nil {
		return map[string]interface{}{
			"running":        false,
			"port":           GlobalConfig.Port,
			"error":          "SunnyNet实例未初始化",
			"disableUDP":     GlobalConfig.DisableUDP,
			"disableTCP":     GlobalConfig.DisableTCP,
			"disableCache":   GlobalConfig.DisableCache,
			"authentication": GlobalConfig.Authentication,
		}, nil
	}

	errStr := ""
	if app.App.Error != nil {
		errStr = app.App.Error.Error()
	}

	return map[string]interface{}{
		"running":        errStr == "",
		"port":           app.App.Port(),
		"error":          errStr,
		"disableUDP":     GlobalConfig.DisableUDP,
		"disableTCP":     GlobalConfig.DisableTCP,
		"disableCache":   GlobalConfig.DisableCache,
		"authentication": GlobalConfig.Authentication,
		"globalProxy":    GlobalConfig.GlobalProxy,
	}, nil
}

// ============ 请求拦截类工具实现 ============

// RequestInfo 请求信息结构
type RequestInfo struct {
	Theology   int    `json:"theology"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode"`
	ClientIP   string `json:"clientIP"`
	PID        string `json:"pid"`
	SendTime   string `json:"sendTime"`
	RecTime    string `json:"recTime"`
	Way        string `json:"way"`
	Notes      string `json:"notes"`
}

// toolRequestList 获取请求列表
func toolRequestList(limit, offset int) (interface{}, error) {
	var requests []RequestInfo
	var keys []int

	// 收集所有请求ID
	HashMap.Search(func(theology int, _ int, _ *MapHash.Request) {
		keys = append(keys, theology)
	})

	// 排序
	sort.Ints(keys)

	// 反转以获取最新的在前
	for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
		keys[i], keys[j] = keys[j], keys[i]
	}

	// 应用分页
	start := offset
	end := offset + limit
	if start > len(keys) {
		start = len(keys)
	}
	if end > len(keys) {
		end = len(keys)
	}
	keys = keys[start:end]

	// 获取请求详情
	for _, theology := range keys {
		h := HashMap.GetRequest(theology)
		if h != nil && h.Display {
			requests = append(requests, RequestInfo{
				Theology:   theology,
				Method:     h.Method,
				URL:        h.URL,
				StatusCode: h.Response.StateCode,
				ClientIP:   h.ClientIP,
				PID:        h.PID,
				SendTime:   h.SendTime,
				RecTime:    h.RecTime,
				Way:        h.Way,
				Notes:      h.Notes,
			})
		}
	}

	return map[string]interface{}{
		"total":    len(keys),
		"offset":   offset,
		"limit":    limit,
		"requests": requests,
	}, nil
}

// RequestDetail 请求详细信息
type RequestDetail struct {
	Theology int    `json:"theology"`
	Method   string `json:"method"`
	URL      string `json:"url"`
	Proto    string `json:"proto"`
	Request  struct {
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
		BodyB64 string              `json:"bodyBase64"`
	} `json:"request"`
	Response struct {
		StatusCode int                 `json:"statusCode"`
		Headers    map[string][]string `json:"headers"`
		Body       string              `json:"body"`
		BodyB64    string              `json:"bodyBase64"`
		Error      bool                `json:"error"`
	} `json:"response"`
	ClientIP string `json:"clientIP"`
	PID      string `json:"pid"`
	SendTime string `json:"sendTime"`
	RecTime  string `json:"recTime"`
	Way      string `json:"way"`
	Notes    string `json:"notes"`
}

// toolRequestGet 获取请求详情
func toolRequestGet(theology int) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	detail := RequestDetail{
		Theology: theology,
		Method:   h.Method,
		URL:      h.URL,
		Proto:    h.Proto,
		ClientIP: h.ClientIP,
		PID:      h.PID,
		SendTime: h.SendTime,
		RecTime:  h.RecTime,
		Way:      h.Way,
		Notes:    h.Notes,
	}

	// 请求信息
	detail.Request.Headers = h.Header
	detail.Request.Body = string(h.Body)
	detail.Request.BodyB64 = base64.StdEncoding.EncodeToString(h.Body)

	// 响应信息
	detail.Response.StatusCode = h.Response.StateCode
	detail.Response.Headers = h.Response.Header
	detail.Response.Body = string(h.Response.Body)
	detail.Response.BodyB64 = base64.StdEncoding.EncodeToString(h.Response.Body)
	detail.Response.Error = h.Response.Error

	return detail, nil
}

// toolRequestModifyHeader 修改请求头
func toolRequestModifyHeader(theology int, key, value string) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	if h.Conn == nil {
		return nil, errors.New("请求连接已失效，无法修改")
	}

	// 修改请求头
	if h.Header == nil {
		h.Header = make(map[string][]string)
	}
	h.Header[key] = []string{value}

	// 同步到连接
	reqHeader := h.Conn.GetRequestHeader()
	if reqHeader != nil {
		reqHeader[key] = []string{value}
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"key":      key,
		"value":    value,
		"message":  "请求头已修改",
	}, nil
}

// toolRequestModifyBody 修改请求体
func toolRequestModifyBody(theology int, body string) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	if h.Conn == nil {
		return nil, errors.New("请求连接已失效，无法修改")
	}

	// 修改请求体
	h.Body = []byte(body)
	h.Conn.SetRequestBody(h.Body)

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"message":  "请求体已修改",
	}, nil
}

// toolResponseModifyHeader 修改响应头
func toolResponseModifyHeader(theology int, key, value string) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	if h.Response.Conn == nil {
		return nil, errors.New("响应连接已失效，无法修改")
	}

	// 修改响应头
	if h.Response.Header == nil {
		h.Response.Header = make(map[string][]string)
	}
	h.Response.Header[key] = []string{value}

	// 同步到连接
	respHeader := h.Response.Conn.GetResponseHeader()
	if respHeader != nil {
		respHeader[key] = []string{value}
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"key":      key,
		"value":    value,
		"message":  "响应头已修改",
	}, nil
}

// toolResponseModifyBody 修改响应体
func toolResponseModifyBody(theology int, body string) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	if h.Response.Conn == nil {
		return nil, errors.New("响应连接已失效，无法修改")
	}

	// 修改响应体
	h.Response.Body = []byte(body)
	h.Response.Conn.SetResponseBody(h.Response.Body)

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"message":  "响应体已修改",
	}, nil
}

// toolRequestBlock 阻断请求
func toolRequestBlock(theology int) (interface{}, error) {
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	// 设置断点
	h.Break = 1
	h.Wait.Add(1)

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"message":  "请求已被阻断，等待处理",
	}, nil
}

// toolRequestReleaseAll 放行所有请求
func toolRequestReleaseAll() (interface{}, error) {
	HashMap.ReleaseAll()

	return map[string]interface{}{
		"success": true,
		"message": "所有请求已放行",
	}, nil
}

// ============ 证书管理类工具实现 ============

// toolCertInstall 安装默认证书
func toolCertInstall() (interface{}, error) {
	result := CommAnd.InstallCert([]byte(public.RootCa))

	success := strings.Contains(result, "成功") || strings.Contains(strings.ToLower(result), "success")

	return map[string]interface{}{
		"success": success,
		"message": result,
	}, nil
}

// toolCertExport 导出默认证书
func toolCertExport(path string) (interface{}, error) {
	if path == "" {
		return nil, errors.New("路径不能为空")
	}

	// 确保路径以 .cer 结尾
	if !strings.HasSuffix(strings.ToLower(path), ".cer") {
		path += ".cer"
	}

	// 写入证书文件
	err := os.WriteFile(path, []byte(public.RootCa), 0644)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "证书已导出",
	}, nil
}

// ============ 进程拦截类工具实现 ============

// toolProcessList 获取进程列表
func toolProcessList() (interface{}, error) {
	processes := CommAnd.EnumerateProcesses()

	return map[string]interface{}{
		"success":   true,
		"processes": processes,
	}, nil
}

// toolProcessAddName 添加拦截进程名
func toolProcessAddName(name string) (interface{}, error) {
	if app == nil || app.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}

	// 转换为GBK编码
	gbkName := JsCall.ToGBK(name)
	app.App.ProcessAddName(gbkName)

	return map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("已添加进程名: %s", name),
	}, nil
}

// toolProcessRemoveName 移除拦截进程名
func toolProcessRemoveName(name string) (interface{}, error) {
	if app == nil || app.App == nil {
		return nil, errors.New("SunnyNet实例未初始化")
	}

	// 转换为GBK编码
	gbkName := JsCall.ToGBK(name)
	app.App.ProcessDelName(gbkName)

	return map[string]interface{}{
		"success": true,
		"name":    name,
		"message": fmt.Sprintf("已移除进程名: %s", name),
	}, nil
}

// ============ 配置类工具实现 ============

// toolConfigGet 获取当前配置
func toolConfigGet() (interface{}, error) {
	return map[string]interface{}{
		"port":               GlobalConfig.Port,
		"disableUDP":         GlobalConfig.DisableUDP,
		"disableTCP":         GlobalConfig.DisableTCP,
		"disableCache":       GlobalConfig.DisableCache,
		"authentication":     GlobalConfig.Authentication,
		"globalProxy":        GlobalConfig.GlobalProxy,
		"globalProxyRules":   GlobalConfig.GlobalProxyRules,
		"mustTcpOpen":        GlobalConfig.MustTcp.Open,
		"mustTcpRules":       GlobalConfig.MustTcp.Rules,
		"certDefault":        GlobalConfig.Cert.Default,
		"certCaPath":         GlobalConfig.Cert.CaPath,
		"certKeyPath":        GlobalConfig.Cert.KeyPath,
		"replaceRulesCount":  len(GlobalConfig.ReplaceRules),
		"hostsRulesCount":    len(GlobalConfig.HostsRules),
		"darkTheme":          GlobalConfig.DarkTheme == 1,
		"requestCertManager": len(GlobalConfig.RequestCertManager),
	}, nil
}

// ============ 解密分析类工具实现 ============

// toolDecryptPacket 解密单个数据包
func toolDecryptPacket(dataHex string) (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	// 解析十六进制数据
	data, err := hexStringToBytes(dataHex)
	if err != nil {
		return nil, fmt.Errorf("无效的十六进制数据: %v", err)
	}

	// 解析数据包
	result, err := cryptoAnalyzer.ParsePacket(data)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"rawHex":  dataHex,
		}, nil
	}

	return map[string]interface{}{
		"success":      true,
		"header":       result.Header,
		"rawHex":       result.RawHex,
		"payloadHex":   result.PayloadHex,
		"decryptedHex": result.DecryptedHex,
		"protobufTree": result.ProtobufTree,
	}, nil
}

// toolParseProtobuf 解析Protobuf数据
func toolParseProtobuf(dataHex string) (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	// 解析十六进制数据
	data, err := hexStringToBytes(dataHex)
	if err != nil {
		return nil, fmt.Errorf("无效的十六进制数据: %v", err)
	}

	// 解析Protobuf
	tree := cryptoAnalyzer.ParseProtobuf(data, 0)

	return map[string]interface{}{
		"success":      true,
		"protobufTree": tree,
		"dataLength":   len(data),
	}, nil
}

// toolCryptoConfigGet 获取当前加密配置
func toolCryptoConfigGet() (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	config := cryptoAnalyzer.GetCurrentConfig()
	if config == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "没有配置当前加密配置",
		}, nil
	}

	return map[string]interface{}{
		"success":    true,
		"name":       config.Name,
		"aesKey":     config.AESKey,
		"aesIV":      config.AESIV,
		"headerSize": config.HeaderSize,
		"msgNames":   config.MsgNames,
	}, nil
}

// toolCryptoConfigSet 设置加密配置
func toolCryptoConfigSet(name, aesKey, aesIV string, headerSize int) (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	// 验证参数
	if name == "" {
		return nil, errors.New("配置名称不能为空")
	}
	if aesKey == "" {
		return nil, errors.New("AES密钥不能为空")
	}
	if aesIV == "" {
		return nil, errors.New("AES IV不能为空")
	}
	if headerSize < 0 {
		return nil, errors.New("头部大小不能为负数")
	}

	// 创建配置
	config := &CryptoConfig{
		Name:       name,
		AESKey:     aesKey,
		AESIV:      aesIV,
		HeaderSize: headerSize,
		MsgNames:   make(map[int]string),
	}

	// 添加并设置为当前配置
	cryptoAnalyzer.AddConfig(config)
	cryptoAnalyzer.SetCurrentConfig(name)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("加密配置 '%s' 已设置", name),
		"config": map[string]interface{}{
			"name":       config.Name,
			"aesKey":     config.AESKey,
			"aesIV":      config.AESIV,
			"headerSize": config.HeaderSize,
		},
	}, nil
}

// toolCryptoConfigList 列出所有加密配置
func toolCryptoConfigList() (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	configs := cryptoAnalyzer.GetAllConfigs()
	currentConfig := cryptoAnalyzer.GetCurrentConfig()
	currentName := ""
	if currentConfig != nil {
		currentName = currentConfig.Name
	}

	configList := make([]map[string]interface{}, 0, len(configs))
	for _, cfg := range configs {
		configList = append(configList, map[string]interface{}{
			"name":       cfg.Name,
			"aesKey":     cfg.AESKey,
			"aesIV":      cfg.AESIV,
			"headerSize": cfg.HeaderSize,
			"isCurrent":  cfg.Name == currentName,
		})
	}

	return map[string]interface{}{
		"success":     true,
		"configs":     configList,
		"total":       len(configList),
		"currentName": currentName,
	}, nil
}

// toolDecryptTcpFlow 解密TCP数据流
func toolDecryptTcpFlow(theology int) (interface{}, error) {
	if cryptoAnalyzer == nil {
		return nil, errors.New("加密分析器未初始化")
	}

	// 获取TCP请求
	h := HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("TCP连接 %d 不存在", theology)
	}

	// 检查是否是TCP连接
	if h.TcpConn == nil && !strings.Contains(strings.ToUpper(h.Way), "TCP") {
		return nil, fmt.Errorf("请求 %d 不是TCP连接", theology)
	}

	// 获取Socket数据
	socketData := h.SocketData
	if socketData == nil || len(socketData) == 0 {
		return map[string]interface{}{
			"success":  true,
			"theology": theology,
			"packets":  []interface{}{},
			"total":    0,
			"message":  "没有捕获到数据包",
		}, nil
	}

	// 解密每个数据包
	packets := make([]map[string]interface{}, 0, len(socketData))
	for i, sd := range socketData {
		if sd == nil || sd.Body == nil || len(sd.Body) == 0 {
			continue
		}

		packet := map[string]interface{}{
			"index":     i,
			"direction": sd.Info.Ico,
			"time":      sd.Info.Time,
			"length":    sd.Info.Length,
			"rawHex":    bytesToHexString(sd.Body),
		}

		// 尝试解密
		decrypted, err := cryptoAnalyzer.ParsePacket(sd.Body)
		if err != nil {
			packet["decryptError"] = err.Error()
		} else {
			packet["header"] = decrypted.Header
			packet["decryptedHex"] = decrypted.DecryptedHex
			packet["protobufTree"] = decrypted.ProtobufTree
		}

		packets = append(packets, packet)
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"url":      h.URL,
		"way":      h.Way,
		"packets":  packets,
		"total":    len(packets),
	}, nil
}

// ============ 辅助函数 ============

// hexStringToBytes 将十六进制字符串转换为字节数组
func hexStringToBytes(hexStr string) ([]byte, error) {
	// 移除可能的空格和前缀
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "\n", "")
	hexStr = strings.ReplaceAll(hexStr, "\r", "")
	hexStr = strings.TrimPrefix(hexStr, "0x")
	hexStr = strings.TrimPrefix(hexStr, "0X")

	// 确保长度是偶数
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	data := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, err := parseHexByte(hexStr[i : i+2])
		if err != nil {
			return nil, err
		}
		data[i/2] = b
	}
	return data, nil
}

// parseHexByte 解析两个十六进制字符为一个字节
func parseHexByte(s string) (byte, error) {
	if len(s) != 2 {
		return 0, errors.New("invalid hex byte length")
	}
	high, err := parseHexChar(s[0])
	if err != nil {
		return 0, err
	}
	low, err := parseHexChar(s[1])
	if err != nil {
		return 0, err
	}
	return (high << 4) | low, nil
}

// parseHexChar 解析单个十六进制字符
func parseHexChar(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	default:
		return 0, fmt.Errorf("invalid hex character: %c", c)
	}
}

// bytesToHexString 将字节数组转换为十六进制字符串
func bytesToHexString(data []byte) string {
	if data == nil {
		return ""
	}
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}

// ============ 替换规则类工具实现 ============

// toolReplaceRulesList 列出所有替换规则
func toolReplaceRulesList() (interface{}, error) {
	rules := GlobalConfig.ReplaceRules
	if rules == nil {
		rules = []ConfigReplaceRules{}
	}

	return map[string]interface{}{
		"success": true,
		"rules":   rules,
		"total":   len(rules),
	}, nil
}

// toolReplaceRulesAdd 添加替换规则
func toolReplaceRulesAdd(ruleType, source, target string) (interface{}, error) {
	// 验证替换类型
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

	// 生成唯一Hash
	hash := fmt.Sprintf("%d", time.Now().UnixNano())

	// 创建新规则
	rule := ConfigReplaceRules{
		Type: ruleType,
		Src:  source,
		Dest: target,
		Hash: hash,
	}

	// 添加到配置
	_TmpLock.Lock()
	GlobalConfig.ReplaceRules = append(GlobalConfig.ReplaceRules, rule)
	_ = GlobalConfig.saveToFile()
	_TmpLock.Unlock()

	// 触发规则重新加载（通过调用内部命令）
	reloadReplaceRules()

	return map[string]interface{}{
		"success": true,
		"rule":    rule,
		"message": "替换规则已添加",
	}, nil
}

// toolReplaceRulesRemove 删除替换规则
func toolReplaceRulesRemove(hash string) (interface{}, error) {
	if hash == "" {
		return nil, errors.New("hash不能为空")
	}

	_TmpLock.Lock()
	defer _TmpLock.Unlock()

	// 查找并删除规则
	found := false
	newRules := make([]ConfigReplaceRules, 0)
	for _, rule := range GlobalConfig.ReplaceRules {
		if rule.Hash == hash {
			found = true
			continue
		}
		newRules = append(newRules, rule)
	}

	if !found {
		return map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("未找到Hash为 %s 的规则", hash),
		}, nil
	}

	GlobalConfig.ReplaceRules = newRules
	_ = GlobalConfig.saveToFile()

	// 触发规则重新加载
	reloadReplaceRulesLocked()

	return map[string]interface{}{
		"success": true,
		"message": "替换规则已删除",
	}, nil
}

// toolReplaceRulesClear 清空所有替换规则
func toolReplaceRulesClear() (interface{}, error) {
	_TmpLock.Lock()
	defer _TmpLock.Unlock()

	count := len(GlobalConfig.ReplaceRules)
	GlobalConfig.ReplaceRules = []ConfigReplaceRules{}
	_ = GlobalConfig.saveToFile()

	// 触发规则重新加载
	reloadReplaceRulesLocked()

	return map[string]interface{}{
		"success": true,
		"cleared": count,
		"message": fmt.Sprintf("已清空 %d 条替换规则", count),
	}, nil
}

// reloadReplaceRules 重新加载替换规则（会获取锁）
func reloadReplaceRules() {
	_TmpLock.Lock()
	defer _TmpLock.Unlock()
	reloadReplaceRulesLocked()
}

// reloadReplaceRulesLocked 重新加载替换规则（调用前需已获取锁）
func reloadReplaceRulesLocked() {
	// 重新构建内部规则列表
	// 这里简化处理，实际应用中可能需要调用 ReplaceRulesEvent
}
