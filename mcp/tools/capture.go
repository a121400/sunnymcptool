package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a121400/sunnymcptool/mcp"
)

var (
	captureProc   *exec.Cmd
	captureMu     sync.Mutex
	captureFile   string
	captureIface  string
	captureFilter string
	captureStart  time.Time
)

func findTshark() string {
	selfDir, _ := os.Executable()
	if selfDir != "" {
		selfDir = filepath.Dir(selfDir)
	}
	candidates := []string{
		filepath.Join(selfDir, "wireshark", "tshark.exe"),
		filepath.Join(selfDir, "tshark.exe"),
		`C:\Program Files\Wireshark\tshark.exe`,
		`C:\Program Files (x86)\Wireshark\tshark.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("tshark"); err == nil {
		return p
	}
	return ""
}

func findDumpcap() string {
	tshark := findTshark()
	if tshark == "" {
		return ""
	}
	dumpcap := filepath.Join(filepath.Dir(tshark), "dumpcap.exe")
	if _, err := os.Stat(dumpcap); err == nil {
		return dumpcap
	}
	return ""
}

func init() {
	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_list_interfaces",
		Description: "列出可用的网络接口（用于抓包）",
		InputSchema: noParamsSchema(),
		Handler:     captureListInterfacesHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_start",
		Description: "开始在指定网络接口上抓包",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"interface": map[string]interface{}{
					"type":        "string",
					"description": "网络接口名称或编号（从 capture_list_interfaces 获取）",
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "BPF 捕获过滤器（如 'tcp port 80'、'host 192.168.1.1'），可选",
				},
				"duration": map[string]interface{}{
					"type":        "integer",
					"description": "抓包持续时间（秒），0 表示手动停止，默认 30",
				},
				"max_packets": map[string]interface{}{
					"type":        "integer",
					"description": "最大抓包数量，0 表示不限制",
				},
			},
			"required": []string{"interface"},
		},
		Handler: captureStartHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_stop",
		Description: "停止当前正在运行的抓包",
		InputSchema: noParamsSchema(),
		Handler:     captureStopHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_status",
		Description: "获取当前抓包状态",
		InputSchema: noParamsSchema(),
		Handler:     captureStatusHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_read_pcap",
		Description: "读取 pcap 文件并应用显示过滤器，返回数据包摘要",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "pcap/pcapng 文件路径（留空则使用最近一次抓包的文件）",
				},
				"display_filter": map[string]interface{}{
					"type":        "string",
					"description": "Wireshark 显示过滤器（如 'http.request'、'dns'、'tcp.port==443'）",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "跳过前 N 个包，默认 0",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回最多 N 个包，默认 50",
				},
				"fields": map[string]interface{}{
					"type":        "string",
					"description": "自定义输出字段（逗号分隔，如 'ip.src,ip.dst,tcp.port'），留空则使用默认摘要格式",
				},
			},
		},
		Handler: captureReadPcapHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_packet_detail",
		Description: "获取指定数据包的完整协议解析详情",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "pcap 文件路径（留空则使用最近一次抓包的文件）",
				},
				"frame_number": map[string]interface{}{
					"type":        "integer",
					"description": "帧编号（从 capture_read_pcap 结果中获取）",
				},
			},
			"required": []string{"frame_number"},
		},
		Handler: capturePacketDetailHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_decode_tls",
		Description: "使用 TLS 密钥日志文件解密 pcap 中的 HTTPS 流量",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "pcap 文件路径",
				},
				"keylog_file": map[string]interface{}{
					"type":        "string",
					"description": "TLS keylog 文件路径（SSLKEYLOGFILE 格式）",
				},
				"display_filter": map[string]interface{}{
					"type":        "string",
					"description": "显示过滤器（如 'http2'、'http'）",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回最多 N 个包，默认 50",
				},
			},
			"required": []string{"file", "keylog_file"},
		},
		Handler: captureDecodeTlsHandler,
	})

	mcp.GlobalRegistry.Register(mcp.ToolDefinition{
		Name:        "capture_statistics",
		Description: "获取 pcap 文件的统计信息（协议分布、会话、端点等）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "pcap 文件路径（留空则使用最近一次抓包的文件）",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "统计类型: summary(摘要) | protocols(协议层次) | conversations(会话) | endpoints(端点)",
					"enum":        []string{"summary", "protocols", "conversations", "endpoints"},
				},
			},
			"required": []string{"type"},
		},
		Handler: captureStatisticsHandler,
	})
}

func captureListInterfacesHandler(params map[string]interface{}) (interface{}, error) {
	tshark := findTshark()
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark，请确认 Wireshark 已安装")
	}
	cmd := exec.Command(tshark, "-D")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("列出接口失败: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	type ifaceInfo struct {
		Index string `json:"index"`
		Name  string `json:"name"`
		Desc  string `json:"description"`
	}
	var ifaces []ifaceInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ". ", 2)
		if len(parts) == 2 {
			nameDesc := parts[1]
			name := nameDesc
			desc := ""
			if idx := strings.Index(nameDesc, " ("); idx > 0 {
				name = nameDesc[:idx]
				desc = strings.Trim(nameDesc[idx+2:], ")")
			}
			ifaces = append(ifaces, ifaceInfo{Index: parts[0], Name: name, Desc: desc})
		}
	}
	return map[string]interface{}{
		"tshark_path": tshark,
		"interfaces":  ifaces,
	}, nil
}

func captureStartHandler(params map[string]interface{}) (interface{}, error) {
	captureMu.Lock()
	defer captureMu.Unlock()

	if captureProc != nil && captureProc.Process != nil {
		return nil, fmt.Errorf("已有抓包正在运行，请先调用 capture_stop")
	}

	tshark := findDumpcap()
	if tshark == "" {
		tshark = findTshark()
	}
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark/dumpcap")
	}

	iface, _ := params["interface"].(string)
	if iface == "" {
		return nil, fmt.Errorf("必须指定网络接口")
	}

	filter, _ := params["filter"].(string)
	duration := 30
	if d, ok := params["duration"].(float64); ok && d > 0 {
		duration = int(d)
	}
	maxPackets := 0
	if m, ok := params["max_packets"].(float64); ok && m > 0 {
		maxPackets = int(m)
	}

	tmpDir := os.TempDir()
	captureFile = filepath.Join(tmpDir, fmt.Sprintf("sunny_capture_%d.pcapng", time.Now().UnixNano()))

	args := []string{"-i", iface, "-w", captureFile}
	if filter != "" {
		args = append(args, "-f", filter)
	}
	if duration > 0 {
		args = append(args, "-a", fmt.Sprintf("duration:%d", duration))
	}
	if maxPackets > 0 {
		args = append(args, "-c", strconv.Itoa(maxPackets))
	}

	cmd := exec.Command(tshark, args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动抓包失败: %v", err)
	}

	captureProc = cmd
	captureIface = iface
	captureFilter = filter
	captureStart = time.Now()

	go func() {
		_ = cmd.Wait()
		captureMu.Lock()
		captureProc = nil
		captureMu.Unlock()
	}()

	return map[string]interface{}{
		"status":    "capturing",
		"interface": iface,
		"filter":    filter,
		"duration":  duration,
		"file":      captureFile,
	}, nil
}

func captureStopHandler(params map[string]interface{}) (interface{}, error) {
	captureMu.Lock()
	defer captureMu.Unlock()

	if captureProc == nil || captureProc.Process == nil {
		return map[string]interface{}{
			"status": "not_running",
			"file":   captureFile,
		}, nil
	}

	_ = captureProc.Process.Kill()
	captureProc = nil

	fi, _ := os.Stat(captureFile)
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}

	return map[string]interface{}{
		"status":   "stopped",
		"file":     captureFile,
		"size":     size,
		"duration": time.Since(captureStart).Seconds(),
	}, nil
}

func captureStatusHandler(params map[string]interface{}) (interface{}, error) {
	captureMu.Lock()
	defer captureMu.Unlock()

	running := captureProc != nil && captureProc.Process != nil
	result := map[string]interface{}{
		"running":   running,
		"interface": captureIface,
		"filter":    captureFilter,
		"file":      captureFile,
	}
	if running {
		result["elapsed_seconds"] = time.Since(captureStart).Seconds()
	}
	if captureFile != "" {
		if fi, err := os.Stat(captureFile); err == nil {
			result["file_size"] = fi.Size()
		}
	}
	return result, nil
}

func captureReadPcapHandler(params map[string]interface{}) (interface{}, error) {
	tshark := findTshark()
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark")
	}

	file, _ := params["file"].(string)
	if file == "" {
		file = captureFile
	}
	if file == "" {
		return nil, fmt.Errorf("未指定 pcap 文件且无最近抓包")
	}
	if _, err := os.Stat(file); err != nil {
		return nil, fmt.Errorf("文件不存在: %s", file)
	}

	displayFilter, _ := params["display_filter"].(string)
	limit := 50
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	offset := 0
	if o, ok := params["offset"].(float64); ok && o > 0 {
		offset = int(o)
	}
	fields, _ := params["fields"].(string)

	args := []string{"-r", file, "-T", "ek"}
	if displayFilter != "" {
		args = append(args, "-Y", displayFilter)
	}
	if fields != "" {
		args = []string{"-r", file, "-T", "fields"}
		for _, f := range strings.Split(fields, ",") {
			args = append(args, "-e", strings.TrimSpace(f))
		}
		args = append(args, "-E", "header=y", "-E", "separator=\t")
		if displayFilter != "" {
			args = append(args, "-Y", displayFilter)
		}
	}

	cmd := exec.Command(tshark, args...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		return nil, fmt.Errorf("tshark 执行失败: %v\n%s", err, stderr)
	}

	if fields != "" {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		end := offset + limit
		if end > len(lines) {
			end = len(lines)
		}
		result := lines
		if offset < len(lines) {
			result = lines[offset:end]
		}
		return map[string]interface{}{
			"total":   len(lines) - 1,
			"packets": result,
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var packets []map[string]interface{}
	count := 0
	skipped := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if _, ok := obj["index"]; ok {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		packets = append(packets, obj)
		count++
		if count >= limit {
			break
		}
	}

	return map[string]interface{}{
		"file":    file,
		"filter":  displayFilter,
		"count":   len(packets),
		"packets": packets,
	}, nil
}

func capturePacketDetailHandler(params map[string]interface{}) (interface{}, error) {
	tshark := findTshark()
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark")
	}

	file, _ := params["file"].(string)
	if file == "" {
		file = captureFile
	}
	if file == "" {
		return nil, fmt.Errorf("未指定 pcap 文件")
	}

	frameNum, ok := params["frame_number"].(float64)
	if !ok || frameNum < 1 {
		return nil, fmt.Errorf("必须指定有效的帧编号")
	}

	filter := fmt.Sprintf("frame.number==%d", int(frameNum))
	cmd := exec.Command(tshark, "-r", file, "-Y", filter, "-V")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tshark 执行失败: %v", err)
	}

	return map[string]interface{}{
		"frame_number": int(frameNum),
		"detail":       string(out),
	}, nil
}

func captureDecodeTlsHandler(params map[string]interface{}) (interface{}, error) {
	tshark := findTshark()
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark")
	}

	file, _ := params["file"].(string)
	keylog, _ := params["keylog_file"].(string)
	displayFilter, _ := params["display_filter"].(string)
	limit := 50
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	if file == "" || keylog == "" {
		return nil, fmt.Errorf("必须指定 pcap 文件和 keylog 文件")
	}

	args := []string{"-r", file, "-o", fmt.Sprintf("tls.keylog_file:%s", keylog), "-T", "fields",
		"-e", "frame.number", "-e", "frame.time_relative", "-e", "ip.src", "-e", "ip.dst",
		"-e", "tcp.srcport", "-e", "tcp.dstport", "-e", "http.request.method", "-e", "http.request.uri",
		"-e", "http.response.code", "-e", "http2.headers.method", "-e", "http2.headers.path",
		"-E", "header=y", "-E", "separator=\t",
		"-c", strconv.Itoa(limit),
	}
	if displayFilter != "" {
		args = append(args, "-Y", displayFilter)
	}

	cmd := exec.Command(tshark, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tshark TLS解密失败: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return map[string]interface{}{
		"file":    file,
		"keylog":  keylog,
		"filter":  displayFilter,
		"packets": lines,
	}, nil
}

func captureStatisticsHandler(params map[string]interface{}) (interface{}, error) {
	tshark := findTshark()
	if tshark == "" {
		return nil, fmt.Errorf("未找到 tshark")
	}

	file, _ := params["file"].(string)
	if file == "" {
		file = captureFile
	}
	if file == "" {
		return nil, fmt.Errorf("未指定 pcap 文件")
	}

	statsType, _ := params["type"].(string)

	var args []string
	switch statsType {
	case "summary":
		capinfos := filepath.Join(filepath.Dir(tshark), "capinfos.exe")
		if _, err := os.Stat(capinfos); err == nil {
			cmd := exec.Command(capinfos, file)
			out, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("capinfos 执行失败: %v", err)
			}
			return map[string]interface{}{"type": "summary", "info": string(out)}, nil
		}
		args = []string{"-r", file, "-z", "io,stat,0"}
	case "protocols":
		args = []string{"-r", file, "-z", "io,phs", "-q"}
	case "conversations":
		args = []string{"-r", file, "-z", "conv,tcp", "-q"}
	case "endpoints":
		args = []string{"-r", file, "-z", "endpoints,ip", "-q"}
	default:
		return nil, fmt.Errorf("不支持的统计类型: %s", statsType)
	}

	cmd := exec.Command(tshark, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("统计分析失败: %v", err)
	}

	result := map[string]interface{}{
		"type": statsType,
		"file": file,
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	result["output"] = strings.Join(lines, "\n")
	return result, nil
}
