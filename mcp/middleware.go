package mcp

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MiddlewareFunc 中间件函数签名
type MiddlewareFunc func(request JSONRPCRequest, next func(JSONRPCRequest) JSONRPCResponse) JSONRPCResponse

// ChainMiddlewares 将多个中间件串联成处理链
func ChainMiddlewares(middlewares []MiddlewareFunc, final func(JSONRPCRequest) JSONRPCResponse) func(JSONRPCRequest) JSONRPCResponse {
	if len(middlewares) == 0 {
		return final
	}
	handler := final
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		next := handler
		handler = func(req JSONRPCRequest) JSONRPCResponse {
			return mw(req, next)
		}
	}
	return handler
}

// LoggingMiddleware 请求日志中间件
func LoggingMiddleware() MiddlewareFunc {
	return func(request JSONRPCRequest, next func(JSONRPCRequest) JSONRPCResponse) JSONRPCResponse {
		start := time.Now()
		response := next(request)
		duration := time.Since(start)
		status := "成功"
		if response.Error != nil {
			status = fmt.Sprintf("失败(错误码:%d)", response.Error.Code)
		}
		log.Printf("[信息] MCP请求: 方法=%s, 耗时=%v, 状态=%s\n", request.Method, duration, status)
		return response
	}
}

// rateLimitEntry 限流记录
type rateLimitEntry struct {
	count     int       // 当前时间窗口内的请求计数
	resetTime time.Time // 计数重置时间
}

// RateLimitMiddleware 限流中间件
// maxRequests: 时间窗口内最大请求数
// window: 时间窗口持续时间
func RateLimitMiddleware(maxRequests int, window time.Duration) MiddlewareFunc {
	var mu sync.Mutex
	entries := make(map[string]*rateLimitEntry)

	return func(request JSONRPCRequest, next func(JSONRPCRequest) JSONRPCResponse) JSONRPCResponse {
		key := request.Method
		if key == "" {
			key = "_global_"
		}

		mu.Lock()

		now := time.Now()
		entry, exists := entries[key]

		if !exists || now.After(entry.resetTime) {
			entries[key] = &rateLimitEntry{
				count:     1,
				resetTime: now.Add(window),
			}
			mu.Unlock()
			return next(request)
		}

		if entry.count >= maxRequests {
			mu.Unlock()
			log.Printf("[警告] 限流触发: 方法=%s, 当前计数=%d, 最大允许=%d\n",
				request.Method, entry.count, maxRequests)
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "请求过于频繁，请稍后重试",
				},
			}
		}

		entry.count++
		mu.Unlock()

		return next(request)
	}
}
