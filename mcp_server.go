package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// 全局变量
var mcpServer *MCPServer

// MCPServer MCP服务器结构体
type MCPServer struct {
	httpServer *http.Server
	port       int
	running    bool
	mu         sync.RWMutex
	clients    map[string]chan []byte // SSE客户端连接
	clientsMu  sync.RWMutex
}

// JSON-RPC 2.0 请求结构
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id,omitempty"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSON-RPC 2.0 响应结构
type JSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      interface{}      `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// JSON-RPC 2.0 错误结构
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP 协议信息
type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPCapabilities struct {
	Tools     *MCPToolsCapability     `json:"tools,omitempty"`
	Resources *MCPResourcesCapability `json:"resources,omitempty"`
}

type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ServerInfo      MCPServerInfo   `json:"serverInfo"`
}

type MCPToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPToolCallResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewMCPServer 创建新的MCP服务器实例
func NewMCPServer(port int) *MCPServer {
	return &MCPServer{
		port:    port,
		running: false,
		clients: make(map[string]chan []byte),
	}
}

// Start 启动MCP服务器
func (m *MCPServer) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("MCP服务器已在运行")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", m.handleMCP)
	mux.HandleFunc("/mcp/sse", m.handleSSE)
	mux.HandleFunc("/mcp/health", m.handleHealth)
	// 添加根路径的消息端点（用于SSE会话的消息发送）
	mux.HandleFunc("/mcp/message", m.handleMCP)

	m.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", m.port),
		Handler:      m.corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 异步启动服务器
	go func() {
		err := m.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP服务器错误: %v\n", err)
			m.mu.Lock()
			m.running = false
			m.mu.Unlock()
		}
	}()

	m.running = true
	fmt.Printf("MCP服务器已启动，端口: %d\n", m.port)
	return nil
}

// Stop 停止MCP服务器
func (m *MCPServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// 关闭所有SSE客户端连接
	m.clientsMu.Lock()
	for id, ch := range m.clients {
		close(ch)
		delete(m.clients, id)
	}
	m.clientsMu.Unlock()

	if m.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.httpServer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}

	m.running = false
	fmt.Println("MCP服务器已停止")
	return nil
}

// IsRunning 检查服务器是否正在运行
func (m *MCPServer) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetPort 获取服务器端口
func (m *MCPServer) GetPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.port
}

// corsMiddleware CORS中间件
func (m *MCPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth 健康检查端点
func (m *MCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"server":  "SunnyNet-MCP",
		"version": "1.0.0",
		"running": m.IsRunning(),
	})
}

// handleSSE SSE事件流端点
func (m *MCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 检查是否支持刷新
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持SSE", http.StatusInternalServerError)
		return
	}

	// 创建客户端ID和通道
	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	messageChan := make(chan []byte, 100)

	// 注册客户端
	m.clientsMu.Lock()
	m.clients[clientID] = messageChan
	m.clientsMu.Unlock()

	// 清理函数
	defer func() {
		m.clientsMu.Lock()
		delete(m.clients, clientID)
		close(messageChan)
		m.clientsMu.Unlock()
	}()

	// 发送初始连接事件，包含消息端点URL
	messageEndpoint := fmt.Sprintf("http://127.0.0.1:%d/mcp/message?sessionId=%s", m.port, clientID)
	initEvent := fmt.Sprintf("event: endpoint\ndata: %s\n\n", messageEndpoint)
	w.Write([]byte(initEvent))
	flusher.Flush()

	// 保持连接并发送消息
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messageChan:
			if !ok {
				return
			}
			// 发送SSE事件
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msg))
			flusher.Flush()
		case <-time.After(15 * time.Second):
			// 发送心跳保持连接
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// handleMCP JSON-RPC请求端点
func (m *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 解析JSON-RPC请求
	var request JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		m.writeJSONRPCError(w, nil, -32700, "解析错误", err.Error())
		return
	}

	// 验证JSON-RPC版本
	if request.JSONRPC != "2.0" {
		m.writeJSONRPCError(w, request.ID, -32600, "无效的请求", "必须使用JSON-RPC 2.0")
		return
	}

	// 处理请求并获取响应
	response := m.handleJSONRPC(request)

	// 获取sessionId（如果有的话，同时发送SSE响应）
	sessionId := r.URL.Query().Get("sessionId")
	if sessionId != "" {
		m.clientsMu.RLock()
		ch, exists := m.clients[sessionId]
		m.clientsMu.RUnlock()
		if exists {
			respData, _ := json.Marshal(response)
			select {
			case ch <- respData:
				// 同时发送到SSE通道
			default:
				// 通道已满，跳过SSE
			}
		}
	}

	// 始终返回HTTP响应（Cursor可能需要）
	json.NewEncoder(w).Encode(response)
}

// handleJSONRPC 处理JSON-RPC请求
func (m *MCPServer) handleJSONRPC(request JSONRPCRequest) JSONRPCResponse {
	switch request.Method {
	case "initialize":
		return m.handleInitialize(request)
	case "initialized":
		// 客户端确认初始化完成，返回空响应
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  map[string]interface{}{},
		}
	case "tools/list":
		return m.handleToolsList(request)
	case "tools/call":
		return m.handleToolsCall(request)
	case "ping":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  map[string]interface{}{},
		}
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "方法不存在",
				Data:    fmt.Sprintf("未知方法: %s", request.Method),
			},
		}
	}
}

// handleInitialize 处理MCP初始化请求
func (m *MCPServer) handleInitialize(request JSONRPCRequest) JSONRPCResponse {
	result := MCPInitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: MCPCapabilities{
			Tools: &MCPToolsCapability{
				ListChanged: true,
			},
		},
		ServerInfo: MCPServerInfo{
			Name:    "SunnyNet-MCP",
			Version: "1.0.0",
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleToolsList 处理工具列表请求
func (m *MCPServer) handleToolsList(request JSONRPCRequest) JSONRPCResponse {
	tools := GetToolsList()
	result := MCPToolsListResult{
		Tools: tools,
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleToolsCall 处理工具调用请求
func (m *MCPServer) handleToolsCall(request JSONRPCRequest) JSONRPCResponse {
	// 从参数中提取工具名和参数
	params := request.Params
	if params == nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "无效的参数",
				Data:    "缺少必要参数",
			},
		}
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "无效的参数",
				Data:    "缺少工具名称",
			},
		}
	}

	var args map[string]interface{}
	if arguments, ok := params["arguments"].(map[string]interface{}); ok {
		args = arguments
	} else {
		args = make(map[string]interface{})
	}

	// 调用工具
	result, err := CallTool(toolName, args)
	if err != nil {
		// 返回错误结果
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: MCPToolCallResult{
				Content: []MCPContent{
					{
						Type: "text",
						Text: fmt.Sprintf("错误: %v", err),
					},
				},
				IsError: true,
			},
		}
	}

	// 将结果转换为JSON字符串
	resultJSON, err := json.Marshal(result)
	if err != nil {
		resultJSON = []byte(fmt.Sprintf("%v", result))
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: MCPToolCallResult{
			Content: []MCPContent{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
			IsError: false,
		},
	}
}

// writeJSONRPCError 写入JSON-RPC错误响应
func (m *MCPServer) writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	json.NewEncoder(w).Encode(response)
}

// SendToClient 向指定客户端发送消息
func (m *MCPServer) SendToClient(clientID string, message []byte) bool {
	m.clientsMu.RLock()
	ch, exists := m.clients[clientID]
	m.clientsMu.RUnlock()

	if !exists {
		return false
	}

	select {
	case ch <- message:
		return true
	default:
		return false
	}
}

// BroadcastToAll 向所有客户端广播消息
func (m *MCPServer) BroadcastToAll(message []byte) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	for _, ch := range m.clients {
		select {
		case ch <- message:
		default:
			// 通道已满，跳过
		}
	}
}
