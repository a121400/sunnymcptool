package tools

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/a121400/sunnymcptool/MapHash"
	"github.com/a121400/sunnymcptool/mcp"
)

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "socket_data_list",
		Description: "获取 WebSocket/TCP/UDP 连接的数据包列表（消息帧列表），返回每个数据包的方向、大小、时间等摘要",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "连接的唯一ID (Theology)"},
				"limit":    map[string]interface{}{"type": "integer", "description": "返回的最大数量，默认100", "default": 100},
				"offset":   map[string]interface{}{"type": "integer", "description": "偏移量，用于分页", "default": 0},
			},
			"required": []string{"theology"},
		},
		Handler: toolSocketDataListHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "socket_data_get",
		Description: "获取 WebSocket/TCP/UDP 连接的指定数据包详情，包含完整的 Body 内容（文本或Base64）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "连接的唯一ID (Theology)"},
				"index":    map[string]interface{}{"type": "integer", "description": "数据包索引（从0开始）"},
			},
			"required": []string{"theology", "index"},
		},
		Handler: toolSocketDataGetHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "socket_data_get_range",
		Description: "批量获取 WebSocket/TCP/UDP 连接的多个数据包详情，适用于分析连续的协议交互",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"theology": map[string]interface{}{"type": "integer", "description": "连接的唯一ID (Theology)"},
				"start":    map[string]interface{}{"type": "integer", "description": "起始索引（从0开始）"},
				"end":      map[string]interface{}{"type": "integer", "description": "结束索引（不含），默认 start+20"},
			},
			"required": []string{"theology", "start"},
		},
		Handler: toolSocketDataGetRangeHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "connection_list",
		Description: "列出所有 WebSocket/TCP/UDP 长连接，按协议类型分类，显示连接状态和消息数",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"protocol": map[string]interface{}{"type": "string", "description": "协议过滤: websocket/tcp/udp，不填则返回全部"},
				"limit":    map[string]interface{}{"type": "integer", "description": "返回的最大数量，默认50", "default": 50},
				"offset":   map[string]interface{}{"type": "integer", "description": "偏移量，用于分页", "default": 0},
			},
			"required": []string{},
		},
		Handler: toolConnectionListHandler,
	})
}

func toolSocketDataListHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	limit := v.OptionalInt("limit", 100)
	offset := v.OptionalInt("offset", 0)
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := ctx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if len(h.SocketData) == 0 {
		return map[string]interface{}{
			"success": true, "theology": theology, "protocol": h.Way,
			"url": h.URL, "total": 0, "packets": []interface{}{},
		}, nil
	}

	total := len(h.SocketData)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var packets []map[string]interface{}
	for i := start; i < end; i++ {
		sd := h.SocketData[i]
		if sd == nil {
			continue
		}
		pkt := map[string]interface{}{
			"index":  i,
			"length": len(sd.Body),
		}
		if sd.Info != nil {
			pkt["direction"] = sd.Info.Ico
			pkt["time"] = sd.Info.Time
			pkt["type"] = sd.Info.WsType
		}
		if sd.Body != nil && len(sd.Body) > 0 {
			if isPrintable(sd.Body) {
				previewBytes := sd.Body
				suffix := ""
				if len(previewBytes) > 200 {
					previewBytes = previewBytes[:200]
					suffix = "..."
				}
				pkt["preview"] = string(previewBytes) + suffix
			} else {
				pkt["preview"] = fmt.Sprintf("[binary %d bytes]", len(sd.Body))
			}
		}
		packets = append(packets, pkt)
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"protocol": h.Way,
		"url":      h.URL,
		"total":    total,
		"sendNum":  h.SendNum,
		"recNum":   h.RecNum,
		"offset":   offset,
		"limit":    limit,
		"packets":  packets,
	}, nil
}

func toolSocketDataGetHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	index := v.RequireInt("index")
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := ctx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	if len(h.SocketData) == 0 {
		return nil, fmt.Errorf("请求 %d 没有数据包", theology)
	}
	if index < 0 || index >= len(h.SocketData) {
		return nil, fmt.Errorf("索引 %d 超出范围 (0-%d)", index, len(h.SocketData)-1)
	}

	sd := h.SocketData[index]
	if sd == nil {
		return nil, fmt.Errorf("数据包 %d 为空", index)
	}

	pkt := map[string]interface{}{
		"index":  index,
		"length": len(sd.Body),
	}
	if sd.Info != nil {
		pkt["direction"] = sd.Info.Ico
		pkt["time"] = sd.Info.Time
		pkt["type"] = sd.Info.WsType
	}
	if sd.Body != nil {
		if isPrintable(sd.Body) {
			pkt["body"] = string(sd.Body)
		} else {
			pkt["body"] = "[binary]"
		}
		pkt["bodyBase64"] = base64.StdEncoding.EncodeToString(sd.Body)
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"protocol": h.Way,
		"packet":   pkt,
	}, nil
}

func toolSocketDataGetRangeHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	theology := v.RequireInt("theology")
	start := v.RequireInt("start")
	end := v.OptionalInt("end", start+20)
	if err := v.Error(); err != nil {
		return nil, err
	}

	h := ctx().HashMap.GetRequest(theology)
	if h == nil {
		return nil, fmt.Errorf("请求 %d 不存在", theology)
	}
	total := len(h.SocketData)
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start >= total {
		return map[string]interface{}{
			"success": true, "theology": theology, "total": total, "packets": []interface{}{},
		}, nil
	}

	var packets []map[string]interface{}
	for i := start; i < end; i++ {
		sd := h.SocketData[i]
		if sd == nil {
			continue
		}
		pkt := map[string]interface{}{
			"index":  i,
			"length": len(sd.Body),
		}
		if sd.Info != nil {
			pkt["direction"] = sd.Info.Ico
			pkt["time"] = sd.Info.Time
			pkt["type"] = sd.Info.WsType
		}
		if sd.Body != nil {
			if isPrintable(sd.Body) {
				pkt["body"] = string(sd.Body)
			} else {
				pkt["body"] = "[binary]"
			}
			pkt["bodyBase64"] = base64.StdEncoding.EncodeToString(sd.Body)
		}
		packets = append(packets, pkt)
	}

	return map[string]interface{}{
		"success":  true,
		"theology": theology,
		"protocol": h.Way,
		"total":    total,
		"range":    []int{start, end},
		"packets":  packets,
	}, nil
}

func toolConnectionListHandler(args map[string]interface{}) (interface{}, error) {
	v := mcp.NewParamValidator(args)
	protocolFilter := v.OptionalString("protocol", "")
	limit := v.OptionalInt("limit", 50)
	offset := v.OptionalInt("offset", 0)
	if err := v.Error(); err != nil {
		return nil, err
	}

	protocolFilter = strings.ToLower(protocolFilter)
	hashMap := ctx().HashMap
	var matchedKeys []int

	hashMap.Search(func(theology int, _ int, req *MapHash.Request) {
		if req == nil || !req.Display {
			return
		}
		way := strings.ToLower(req.Way)
		isSocket := way == "websocket" || strings.Contains(way, "tcp") || way == "udp"
		if !isSocket {
			return
		}
		if protocolFilter != "" {
			if protocolFilter == "websocket" && way != "websocket" {
				return
			}
			if protocolFilter == "tcp" && !strings.Contains(way, "tcp") {
				return
			}
			if protocolFilter == "udp" && way != "udp" {
				return
			}
		}
		matchedKeys = append(matchedKeys, theology)
	})

	paged, totalMatched := sortDescAndPaginate(matchedKeys, offset, limit)

	var connections []map[string]interface{}
	for _, theology := range paged {
		h := hashMap.GetRequest(theology)
		if h == nil {
			continue
		}
		status := "已断开"
		if h.WsConn != nil || h.TcpConn != nil || h.UdpConn != nil {
			status = "已连接"
		}
		connections = append(connections, map[string]interface{}{
			"theology":    theology,
			"protocol":    h.Way,
			"url":         h.URL,
			"status":      status,
			"sendNum":     h.SendNum,
			"recNum":      h.RecNum,
			"packetCount": len(h.SocketData),
			"pid":         h.PID,
			"clientIP":    h.ClientIP,
			"sendTime":    h.SendTime,
			"notes":       h.Notes,
		})
	}

	return map[string]interface{}{
		"success":     true,
		"total":       totalMatched,
		"offset":      offset,
		"limit":       limit,
		"connections": connections,
	}, nil
}

func isPrintable(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	nonPrintable := 0
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	for _, b := range check {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(check)) < 0.1
}
