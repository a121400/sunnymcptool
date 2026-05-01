package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type jsonrpcMsg struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
}

func main() {
	log.SetOutput(os.Stderr)

	port := "29999"
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "-port" {
			port = os.Args[i+1]
		}
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%s/mcp", port)
	client := &http.Client{Timeout: 30 * time.Second}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var msg jsonrpcMsg
		json.Unmarshal(line, &msg)

		switch msg.Method {
		case "initialize":
			sendJSON(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      msg.ID,
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]interface{}{
						"name":    "sunnynet",
						"version": "1.0.0",
					},
				},
			})
			continue

		case "notifications/initialized", "initialized":
			continue
		}

		resp, err := client.Post(endpoint, "application/json", bytes.NewReader(line))
		if err != nil {
			writeError(msg.ID, -32603, fmt.Sprintf("连接SunnyNet失败: %v", err))
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		trimmed := strings.TrimRight(string(body), "\r\n ")
		if len(trimmed) > 0 {
			fmt.Fprintln(os.Stdout, trimmed)
		}
	}
}

func sendJSON(v interface{}) {
	b, _ := json.Marshal(v)
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
}

func writeError(id interface{}, code int, msg string) {
	sendJSON(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]interface{}{"code": code, "message": msg},
	})
}
