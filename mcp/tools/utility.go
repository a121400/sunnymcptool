package tools

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_clear",
		Description: "清空全部已捕获的请求列表（保留活跃的长连接）",
		InputSchema: noParamsSchema(),
		Handler:     toolRequestClearHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_delete",
		Description: "删除指定的请求记录",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theologies": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "integer"},
					"description": "要删除的请求ID列表",
				},
			},
			"required": []string{"theologies"},
		},
		Handler: toolRequestDeleteHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_resend",
		Description: "重发指定的 HTTP 请求（当前仅支持 HTTP 协议）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theologies": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "integer"},
					"description": "要重发的请求ID列表",
				},
			},
			"required": []string{"theologies"},
		},
		Handler: toolRequestResendHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_stats",
		Description: "获取当前抓包会话的统计信息：请求总数、协议分布、状态码分布等",
		InputSchema: noParamsSchema(),
		Handler:     toolRequestStatsHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_body_search",
		Description: "在所有已捕获请求的 URL 或 Body 中搜索关键词，返回匹配的请求列表",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{"type": "string", "description": "搜索关键词"},
				"scope":   map[string]interface{}{"type": "string", "description": "搜索范围: url/request_body/response_body/all，默认all", "default": "all"},
				"limit":   map[string]interface{}{"type": "integer", "description": "返回最大数量，默认20", "default": 20},
			},
			"required": []string{"keyword"},
		},
		Handler: toolRequestBodySearchHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_get_response_body_decoded",
		Description: "获取指定请求的响应体，自动处理编码，返回可读文本。对于二进制内容返回Base64",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
			},
			"required": []string{"theology"},
		},
		Handler: toolRequestGetResponseBodyDecodedHandler,
	})
}

func toolRequestClearHandler(args map[string]interface{}) (interface{}, error) {
	ctx().HashMap.Empty()
	return map[string]interface{}{
		"success": true,
		"message": "已清空请求列表（活跃长连接已保留）",
	}, nil
}

func toolRequestDeleteHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theologies := v.RequireIntArray("theologies")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if len(theologies) == 0 {
		return nil, fmt.Errorf("请求ID列表不能为空")
	}
	ctx().HashMap.Delete(theologies)
	return map[string]interface{}{
		"success": true,
		"deleted": len(theologies),
		"message": fmt.Sprintf("已删除 %d 条请求", len(theologies)),
	}, nil
}

func toolRequestResendHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theologies := v.RequireIntArray("theologies")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if len(theologies) == 0 {
		return nil, fmt.Errorf("请求ID列表不能为空")
	}
	c := ctx()
	port := c.App.GetPort()
	c.HashMap.Resend(theologies, 0, port)
	return map[string]interface{}{
		"success": true,
		"count":   len(theologies),
		"message": fmt.Sprintf("已发起 %d 个请求的重发", len(theologies)),
	}, nil
}

func toolRequestStatsHandler(args map[string]interface{}) (interface{}, error) {
	hashMap := ctx().HashMap
	total := 0
	byProtocol := map[string]int{}
	byStatus := map[string]int{}

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req == nil || !req.Display {
			return
		}
		total++
		way := req.Way
		if way == "" {
			way = "HTTP"
		}
		byProtocol[way]++

		sc := req.Response.StateCode
		if sc >= 200 && sc < 300 {
			byStatus["2xx"]++
		} else if sc >= 300 && sc < 400 {
			byStatus["3xx"]++
		} else if sc >= 400 && sc < 500 {
			byStatus["4xx"]++
		} else if sc >= 500 {
			byStatus["5xx"]++
		} else if sc == 0 {
			byStatus["pending"]++
		}
	})

	return map[string]interface{}{
		"success":    true,
		"total":      total,
		"byProtocol": byProtocol,
		"byStatus":   byStatus,
	}, nil
}

func toolRequestBodySearchHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	keyword := v.RequireString("keyword")
	scope := v.OptionalString("scope", "all")
	limit := v.OptionalInt("limit", 20)
	if err := v.Error(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	hashMap := ctx().HashMap
	var matchedKeys []int

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req == nil || !req.Display {
			return
		}
		matched := false
		if (scope == "all" || scope == "url") && contains(req.URL, keyword) {
			matched = true
		}
		if !matched && (scope == "all" || scope == "request_body") && containsBytes(req.Body, keyword) {
			matched = true
		}
		if !matched && (scope == "all" || scope == "response_body") && containsBytes(req.Response.Body, keyword) {
			matched = true
		}
		if matched {
			matchedKeys = append(matchedKeys, theology)
		}
	})

	sort.Ints(matchedKeys)
	for i, j := 0, len(matchedKeys)-1; i < j; i, j = i+1, j-1 {
		matchedKeys[i], matchedKeys[j] = matchedKeys[j], matchedKeys[i]
	}

	totalMatched := len(matchedKeys)
	if limit > 0 && len(matchedKeys) > limit {
		matchedKeys = matchedKeys[:limit]
	}

	var results []map[string]interface{}
	for _, theology := range matchedKeys {
		h := hashMap.GetRequest(theology)
		if h == nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"theology":   theology,
			"method":     h.Method,
			"url":        h.URL,
			"statusCode": h.Response.StateCode,
			"way":        h.Way,
			"sendTime":   h.SendTime,
		})
	}

	return map[string]interface{}{
		"success":  true,
		"keyword":  keyword,
		"scope":    scope,
		"total":    totalMatched,
		"requests": results,
	}, nil
}

func toolRequestGetResponseBodyDecodedHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := ctx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	body := h.Response.Body
	result := map[string]interface{}{
		"theology":   theology,
		"statusCode": h.Response.StateCode,
		"length":     len(body),
	}

	if len(body) == 0 {
		result["body"] = ""
		result["encoding"] = "empty"
	} else if isPrintable(body) {
		result["body"] = string(body)
		result["encoding"] = "text"
	} else {
		result["bodyBase64"] = base64.StdEncoding.EncodeToString(body)
		result["encoding"] = "base64"
	}

	contentType := ""
	if h.Response.Header != nil {
		ct := h.Response.Header["Content-Type"]
		if len(ct) > 0 {
			contentType = ct[0]
		} else {
			ct = h.Response.Header["content-type"]
			if len(ct) > 0 {
				contentType = ct[0]
			}
		}
	}
	result["contentType"] = contentType

	return result, nil
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsBytes(data []byte, keyword string) bool {
	if len(data) == 0 || len(keyword) == 0 {
		return false
	}
	return containsLower(string(data), keyword)
}
