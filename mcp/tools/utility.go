package tools

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/andybalholm/brotli"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func init() {
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

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_highlight_search",
		Description: "搜索关键词并在UI中高亮匹配的请求行，方便定位",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{"type": "string", "description": "搜索关键词"},
				"type":    map[string]interface{}{"type": "string", "description": "搜索类型: UTF8/GBK/Hex，默认UTF8", "default": "UTF8"},
				"color":   map[string]interface{}{"type": "string", "description": "高亮颜色，默认黄色", "default": "#FFFF00"},
			},
			"required": []string{"keyword"},
		},
		Handler: toolRequestHighlightSearchHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_highlight_clear",
		Description: "清除所有搜索高亮标记",
		InputSchema: noParamsSchema(),
		Handler:     toolRequestHighlightClearHandler,
	})
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
	if c == nil || c.App == nil || c.HashMap == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	port := c.App.GetPort()
	c.HashMap.Resend(theologies, 0, port)
	return map[string]interface{}{
		"success": true,
		"count":   len(theologies),
		"message": fmt.Sprintf("已发起 %d 个请求的重发", len(theologies)),
	}, nil
}

func toolRequestStatsHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.HashMap == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	hashMap := c.HashMap
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

	c := ctx()
	if c == nil || c.HashMap == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	hashMap := c.HashMap
	var matchedKeys []int

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req == nil || !req.Display {
			return
		}
		matched := false
		if (scope == "all" || scope == "url") && containsStr(req.URL, keyword) {
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

	paged, totalMatched := sortDescAndPaginate(matchedKeys, 0, limit)

	var results []map[string]interface{}
	for _, theology := range paged {
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

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	body := h.Response.Body
	contentType := getHeaderValue(h.Response.Header, "Content-Type")
	contentEncoding := strings.ToLower(getHeaderValue(h.Response.Header, "Content-Encoding"))

	result := map[string]interface{}{
		"theology":   theology,
		"statusCode": h.Response.StateCode,
		"length":     len(body),
	}

	if len(body) == 0 {
		result["body"] = ""
		result["encoding"] = "empty"
		result["contentType"] = contentType
		return result, nil
	}

	decoded := decompressBody(body, contentEncoding)
	if contentEncoding != "" && len(decoded) != len(body) {
		result["decompressed"] = true
		result["decompressedLength"] = len(decoded)
	}

	decoded = convertCharset(decoded, contentType)

	if isPrintable(decoded) {
		text := string(decoded)
		if looksLikeJSON(contentType, decoded) {
			if pretty, err := prettyJSON(decoded); err == nil {
				text = pretty
				result["formatted"] = true
			}
		}
		result["body"] = text
		result["encoding"] = "text"
	} else {
		result["bodyBase64"] = base64.StdEncoding.EncodeToString(decoded)
		result["encoding"] = "base64"
	}

	result["contentType"] = contentType
	return result, nil
}

func getHeaderValue(headers map[string][]string, key string) string {
	if headers == nil {
		return ""
	}
	if v := headers[key]; len(v) > 0 {
		return v[0]
	}
	lk := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == lk && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func decompressBody(data []byte, encoding string) []byte {
	if len(data) == 0 {
		return data
	}
	var reader io.Reader
	switch encoding {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return data
		}
		defer r.Close()
		reader = r
	case "br":
		reader = brotli.NewReader(bytes.NewReader(data))
	case "deflate":
		r := flate.NewReader(bytes.NewReader(data))
		defer r.Close()
		reader = r
	default:
		return data
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		return data
	}
	return out
}

func convertCharset(data []byte, contentType string) []byte {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "gbk") || strings.Contains(ct, "gb2312") || strings.Contains(ct, "gb18030") {
		decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder()))
		if err == nil {
			return decoded
		}
	}
	return data
}

func looksLikeJSON(contentType string, data []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return true
	}
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

func prettyJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func toolRequestHighlightSearchHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	keyword := v.RequireString("keyword")
	searchType := v.OptionalString("type", "UTF8")
	color := v.OptionalString("color", "#FFFF00")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if keyword == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	c := ctx()
	if c == nil || c.SearchFunc == nil {
		return nil, fmt.Errorf("搜索功能未初始化")
	}
	result := c.SearchFunc(keyword, searchType, color)
	return map[string]interface{}{
		"success": true,
		"keyword": keyword,
		"type":    searchType,
		"color":   color,
		"result":  result,
		"message": "搜索完成，匹配行已高亮",
	}, nil
}

func toolRequestHighlightClearHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.CancelSearchFunc == nil {
		return nil, fmt.Errorf("搜索功能未初始化")
	}
	cleared := c.CancelSearchFunc()
	return map[string]interface{}{
		"success": true,
		"cleared": len(cleared),
		"message": fmt.Sprintf("已清除 %d 条搜索高亮", len(cleared)),
	}, nil
}

func containsStr(s, substr string) bool {
	return len(substr) > 0 && strings.Contains(s, substr)
}

func containsBytes(data []byte, keyword string) bool {
	return len(data) > 0 && len(keyword) > 0 && bytes.Contains(data, []byte(keyword))
}
