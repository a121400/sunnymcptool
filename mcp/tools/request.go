package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"
)

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

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_list",
		Description: "获取已捕获的HTTP请求列表",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit":  map[string]interface{}{"type": "integer", "description": "返回的最大数量，默认100", "default": 100},
				"offset": map[string]interface{}{"type": "integer", "description": "偏移量，用于分页", "default": 0},
			},
			"required": []string{},
		},
		Handler: toolRequestListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_get",
		Description: "获取指定请求的详细信息，包括请求头、请求体、响应头、响应体等",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
			},
			"required": []string{"theology"},
		},
		Handler: toolRequestGetHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_modify_header",
		Description: "修改指定请求的请求头",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
				"key":      map[string]interface{}{"type": "string", "description": "请求头的名称"},
				"value":    map[string]interface{}{"type": "string", "description": "请求头的值"},
			},
			"required": []string{"theology", "key", "value"},
		},
		Handler: toolRequestModifyHeaderHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_modify_body",
		Description: "修改指定请求的请求体",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
				"body":     map[string]interface{}{"type": "string", "description": "新的请求体内容"},
			},
			"required": []string{"theology", "body"},
		},
		Handler: toolRequestModifyBodyHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "response_modify_header",
		Description: "修改指定请求的响应头",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
				"key":      map[string]interface{}{"type": "string", "description": "响应头的名称"},
				"value":    map[string]interface{}{"type": "string", "description": "响应头的值"},
			},
			"required": []string{"theology", "key", "value"},
		},
		Handler: toolResponseModifyHeaderHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "response_modify_body",
		Description: "修改指定请求的响应体",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
				"body":     map[string]interface{}{"type": "string", "description": "新的响应体内容"},
			},
			"required": []string{"theology", "body"},
		},
		Handler: toolResponseModifyBodyHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_block",
		Description: "阻断/拦截指定的请求",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "请求的唯一ID (Theology)"},
			},
			"required": []string{"theology"},
		},
		Handler: toolRequestBlockHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_release_all",
		Description: "放行所有被断点拦截的请求",
		InputSchema: noParamsSchema(),
		Handler:     toolRequestReleaseAllHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_save_all",
		Description: "保存全部已捕获的请求数据到.syn文件（SunnyNet抓包记录格式）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "保存文件路径（如：C:/data/capture.syn）"},
			},
			"required": []string{"path"},
		},
		Handler: toolRequestSaveAllHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_import",
		Description: "从.syn文件导入抓包记录数据到当前列表",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": ".syn文件路径（如：C:/data/capture.syn）"},
			},
			"required": []string{"path"},
		},
		Handler: toolRequestImportHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "request_search",
		Description: "搜索过滤已捕获的请求，支持按URL关键词、HTTP方法、状态码过滤，支持分页",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url":         map[string]interface{}{"type": "string", "description": "URL关键词过滤（包含匹配）"},
				"method":      map[string]interface{}{"type": "string", "description": "HTTP方法过滤（精确匹配，如GET、POST）"},
				"status_code": map[string]interface{}{"type": "integer", "description": "状态码过滤（精确匹配）"},
				"limit":       map[string]interface{}{"type": "integer", "description": "返回的最大数量，默认50", "default": 50},
				"offset":      map[string]interface{}{"type": "integer", "description": "偏移量，用于分页，默认0", "default": 0},
			},
			"required": []string{},
		},
		Handler: toolRequestSearchHandler,
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
		Name:        "request_clear",
		Description: "清空全部已捕获的请求列表（保留活跃的长连接）",
		InputSchema: noParamsSchema(),
		Handler:     toolRequestClearHandler,
	})
}

func toolRequestListHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	limit := v.OptionalInt("limit", 100)
	offset := v.OptionalInt("offset", 0)
	if err := v.Error(); err != nil {
		return nil, err
	}

	c, err := requireCtx()
	if err != nil || c.HashMap == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	hashMap := c.HashMap
	var keys []int

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req != nil && req.Display {
			keys = append(keys, theology)
		}
	})

	paged, total := sortDescAndPaginate(keys, offset, limit)

	requests := make([]RequestInfo, 0, len(paged))
	for _, theology := range paged {
		h := hashMap.GetRequest(theology)
		if h != nil {
			requests = append(requests, RequestInfo{
				Theology: theology, Method: h.Method, URL: h.URL,
				StatusCode: h.Response.StateCode, ClientIP: h.ClientIP,
				PID: h.PID, SendTime: h.SendTime, RecTime: h.RecTime,
				Way: h.Way, Notes: h.Notes,
			})
		}
	}

	return map[string]interface{}{
		"success": true, "total": total, "offset": offset, "limit": limit, "requests": requests,
	}, nil
}

func toolRequestGetHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	if err := v.Error(); err != nil {
		return nil, err
	}

	c, err := requireCtx()
	if err != nil || c.HashMap == nil {
		return nil, errCtxNil
	}
	h := c.HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}

	const maxBodySize = 256 * 1024

	detail := RequestDetail{
		Theology: theology, Method: h.Method, URL: h.URL, Proto: h.Proto,
		ClientIP: h.ClientIP, PID: h.PID, SendTime: h.SendTime, RecTime: h.RecTime,
		Way: h.Way, Notes: h.Notes,
	}
	detail.Request.Headers = h.Header
	detail.Request.Body, detail.Request.BodyB64 = encodeBody(h.Body, maxBodySize)
	detail.Response.StatusCode = h.Response.StateCode
	detail.Response.Headers = h.Response.Header
	detail.Response.Body, detail.Response.BodyB64 = encodeBody(h.Response.Body, maxBodySize)
	detail.Response.Error = h.Response.Error

	return detail, nil
}

func toolRequestModifyHeaderHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	key := v.RequireString("key")
	value := v.RequireString("value")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if h.Conn == nil {
		return nil, errors.New("请求连接已失效，无法修改")
	}
	if h.Header == nil {
		h.Header = make(map[string][]string)
	}
	h.Header[key] = []string{value}
	reqHeader := h.Conn.GetRequestHeader()
	if reqHeader != nil {
		reqHeader[key] = []string{value}
	}

	return map[string]interface{}{
		"success": true, "theology": theology, "key": key, "value": value, "message": "请求头已修改",
	}, nil
}

func toolRequestModifyBodyHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	body := v.RequireString("body")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if h.Conn == nil {
		return nil, errors.New("请求连接已失效，无法修改")
	}
	h.Body = []byte(body)
	h.Conn.SetRequestBody(h.Body)

	return map[string]interface{}{
		"success": true, "theology": theology, "message": "请求体已修改",
	}, nil
}

func toolResponseModifyHeaderHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	key := v.RequireString("key")
	value := v.RequireString("value")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if h.Response.Conn == nil {
		return nil, errors.New("响应连接已失效，无法修改")
	}
	if h.Response.Header == nil {
		h.Response.Header = make(map[string][]string)
	}
	h.Response.Header[key] = []string{value}
	respHeader := h.Response.Conn.GetResponseHeader()
	if respHeader != nil {
		respHeader[key] = []string{value}
	}

	return map[string]interface{}{
		"success": true, "theology": theology, "key": key, "value": value, "message": "响应头已修改",
	}, nil
}

func toolResponseModifyBodyHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	body := v.RequireString("body")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if h.Response.Conn == nil {
		return nil, errors.New("响应连接已失效，无法修改")
	}
	h.Response.Body = []byte(body)
	h.Response.Conn.SetResponseBody(h.Response.Body)

	return map[string]interface{}{
		"success": true, "theology": theology, "message": "响应体已修改",
	}, nil
}

func toolRequestBlockHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := safeCtx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	h.Break = 1

	return map[string]interface{}{
		"success": true, "theology": theology, "message": "请求已被阻断，等待处理",
	}, nil
}

func toolRequestReleaseAllHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.HashMap == nil {
		return nil, fmt.Errorf("应用上下文未初始化")
	}
	c.HashMap.ReleaseAll()
	return map[string]interface{}{
		"success": true, "message": "所有请求已放行",
	}, nil
}

func toolRequestSearchHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	urlFilter := v.OptionalString("url", "")
	methodFilter := v.OptionalString("method", "")
	statusCodeFilter := v.OptionalInt("status_code", 0)
	limit := v.OptionalInt("limit", 50)
	offset := v.OptionalInt("offset", 0)
	if err := v.Error(); err != nil {
		return nil, err
	}

	_, hasStatusCode := args["status_code"]
	c2, err2 := requireCtx()
	if err2 != nil || c2.HashMap == nil {
		return nil, errCtxNil
	}
	hashMap := c2.HashMap
	var matchedKeys []int

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req == nil || !req.Display {
			return
		}
		if urlFilter != "" && !strings.Contains(req.URL, urlFilter) {
			return
		}
		if methodFilter != "" && req.Method != methodFilter {
			return
		}
		if hasStatusCode && statusCodeFilter != 0 && req.Response.StateCode != statusCodeFilter {
			return
		}
		matchedKeys = append(matchedKeys, theology)
	})

	paged, totalMatched := sortDescAndPaginate(matchedKeys, offset, limit)

	var requests []RequestInfo
	for _, theology := range paged {
		h := hashMap.GetRequest(theology)
		if h != nil {
			requests = append(requests, RequestInfo{
				Theology: theology, Method: h.Method, URL: h.URL,
				StatusCode: h.Response.StateCode, ClientIP: h.ClientIP,
				PID: h.PID, SendTime: h.SendTime, RecTime: h.RecTime,
				Way: h.Way, Notes: h.Notes,
			})
		}
	}

	return map[string]interface{}{
		"success": true, "total": totalMatched, "offset": offset, "limit": limit, "requests": requests,
	}, nil
}

func toolRequestSaveAllHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	path := v.RequireString("path")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errors.New("文件路径不能为空")
	}

	cSave, errSave := requireCtx()
	if errSave != nil {
		return nil, errSave
	}
	dio := cSave.DataIO
	if dio == nil {
		return nil, errors.New("数据导入导出模块未初始化")
	}
	if err := dio.SaveAllToFile(path); err != nil {
		return nil, fmt.Errorf("保存失败: %v", err)
	}
	return map[string]interface{}{
		"success": true, "path": path, "message": "全部抓包数据已保存",
	}, nil
}

func toolRequestImportHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	path := v.RequireString("path")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errors.New("文件路径不能为空")
	}

	cImp, errImp := requireCtx()
	if errImp != nil {
		return nil, errImp
	}
	dio := cImp.DataIO
	if dio == nil {
		return nil, errors.New("数据导入导出模块未初始化")
	}
	count, err := dio.ImportFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("导入失败: %v", err)
	}
	return map[string]interface{}{
		"success": true, "path": path, "imported": count, "message": fmt.Sprintf("已导入 %d 条记录", count),
	}, nil
}

func toolRequestDeleteHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theologies := v.RequireIntArray("theologies")
	if err := v.Error(); err != nil {
		return nil, err
	}
	if len(theologies) == 0 {
		return nil, errors.New("请求ID列表不能为空")
	}
	c := ctx()
	if c == nil || c.HashMap == nil {
		return nil, errors.New("应用上下文未初始化")
	}
	c.HashMap.Delete(theologies)
	if c.NotifyUI != nil {
		c.NotifyUI("MCP删除请求", theologies)
	}
	return map[string]interface{}{
		"success": true, "count": len(theologies),
		"message": fmt.Sprintf("已删除 %d 条请求记录", len(theologies)),
	}, nil
}

func toolRequestClearHandler(args map[string]interface{}) (interface{}, error) {
	c := ctx()
	if c == nil || c.HashMap == nil {
		return nil, errors.New("应用上下文未初始化")
	}
	c.HashMap.Empty()
	if c.NotifyUI != nil {
		c.NotifyUI("MCP清空请求", nil)
	}
	return map[string]interface{}{
		"success": true, "message": "已清空全部请求列表",
	}, nil
}
