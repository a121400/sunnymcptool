package tools

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/a121400/sunnymcptool/mcp"
)

// ctx 返回全局应用上下文
func ctx() *mcp.AppContext {
	return mcp.GlobalAppContext
}

// noParamsSchema 返回无参数的输入Schema
func noParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

// sortDescAndPaginate 将 key 切片按降序排列后分页，返回分页结果和原始总数
func sortDescAndPaginate(keys []int, offset, limit int) ([]int, int) {
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	total := len(keys)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	start, end := offset, offset+limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return keys[start:end], total
}

// encodeBody 智能编码 body：可打印文本返回 (text, "")，二进制返回 ("", base64)。
// 超过 maxSize 的部分截断并附加提示。
func encodeBody(data []byte, maxSize int) (text string, b64 string) {
	if len(data) == 0 {
		return "", ""
	}
	truncated := false
	d := data
	if maxSize > 0 && len(d) > maxSize {
		d = d[:maxSize]
		truncated = true
	}
	suffix := ""
	if truncated {
		suffix = fmt.Sprintf("\n... [truncated, total %d bytes]", len(data))
	}
	if isPrintable(d) {
		return string(d) + suffix, ""
	}
	return "", base64.StdEncoding.EncodeToString(d)
}
