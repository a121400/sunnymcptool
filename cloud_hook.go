package main

import (
	"archive/zip"
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/a121400/sunnymcptool/MapHash"
)

//go:embed frida-scripts/frida_inject.mjs
//go:embed frida-scripts/cloudHookAndroid.js
//go:embed frida-scripts/global-shim.js
var fridaScripts embed.FS

type CloudHookState struct {
	mu       sync.Mutex
	running  bool
	stopping bool
	cmd      *exec.Cmd
	device   string
	logs     []string
}

var cloudHook = &CloudHookState{}

type FridaMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type CloudHookMessage struct {
	Type      string `json:"type"`
	ApiType   string `json:"apiType"`
	ID        int    `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Param1    string `json:"param1"`
	Param2    string `json:"param2"`
	Message   string `json:"message"`
}


func (s *CloudHookState) addLog(msg string) {
	s.mu.Lock()
	s.logs = append(s.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	if len(s.logs) > 200 {
		s.logs = s.logs[len(s.logs)-100:]
	}
	s.mu.Unlock()
}

func StartCloudHook() map[string]interface{} {
	cloudHook.mu.Lock()
	if cloudHook.running {
		cloudHook.mu.Unlock()
		return map[string]interface{}{"success": false, "error": "已在运行中"}
	}
	cloudHook.running = true
	cloudHook.stopping = false
	cloudHook.mu.Unlock()

	go runCloudHook()
	return map[string]interface{}{"success": true}
}

func StopCloudHook() map[string]interface{} {
	cloudHook.mu.Lock()
	if !cloudHook.running {
		cloudHook.mu.Unlock()
		return map[string]interface{}{"success": true}
	}
	cloudHook.stopping = true
	if cloudHook.cmd != nil && cloudHook.cmd.Process != nil {
		cloudHook.cmd.Process.Kill()
	}
	cloudHook.mu.Unlock()

	time.Sleep(1 * time.Second)
	cloudHook.mu.Lock()
	cloudHook.running = false
	cloudHook.stopping = false
	cloudHook.cmd = nil
	cloudHook.mu.Unlock()

	return map[string]interface{}{"success": true}
}

func GetCloudHookStatus() map[string]interface{} {
	cloudHook.mu.Lock()
	defer cloudHook.mu.Unlock()
	return map[string]interface{}{
		"running": cloudHook.running,
		"device":  cloudHook.device,
		"logs":    cloudHook.logs,
	}
}

func getFridaScriptsDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	return filepath.Join(appData, "SunnyNet", "frida-scripts")
}

const nodeVersion = "v18.20.8"

func getNodeDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	return filepath.Join(appData, "SunnyNet", "node")
}

func getNodeExe() string {
	return filepath.Join(getNodeDir(), "node.exe")
}

func ensureNodeJS() (string, error) {
	if path, err := exec.LookPath("node"); err == nil {
		return path, nil
	}

	nodeExe := getNodeExe()
	if _, err := os.Stat(nodeExe); err == nil {
		return nodeExe, nil
	}

	CallJsAlert("首次安装", "正在下载 Node.js 运行环境，请稍候...")
	cloudHook.addLog("下载 Node.js 便携版...")
	dir := getNodeDir()
	os.MkdirAll(dir, 0755)

	zipName := fmt.Sprintf("node-%s-win-x64.zip", nodeVersion)
	url := fmt.Sprintf("https://nodejs.org/dist/%s/%s", nodeVersion, zipName)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载 Node.js 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载 Node.js 返回 %d", resp.StatusCode)
	}

	tmpZip := filepath.Join(dir, zipName)
	f, err := os.Create(tmpZip)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return "", err
	}

	CallJsAlert("首次安装", "正在解压 Node.js...")
	cloudHook.addLog("解压 Node.js...")
	if err := unzipNodeToDir(tmpZip, dir); err != nil {
		os.Remove(tmpZip)
		return "", fmt.Errorf("解压 Node.js 失败: %v", err)
	}
	os.Remove(tmpZip)

	if _, err := os.Stat(nodeExe); err != nil {
		return "", fmt.Errorf("解压后找不到 node.exe")
	}

	cloudHook.addLog("Node.js 就绪")
	return nodeExe, nil
}

func unzipNodeToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	prefix := fmt.Sprintf("node-%s-win-x64/", nodeVersion)
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}
		relPath := strings.TrimPrefix(f.Name, prefix)
		if relPath == "" {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(relPath))

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

func findNpm(nodeExe string) string {
	nodeDir := filepath.Dir(nodeExe)
	npmCmd := filepath.Join(nodeDir, "npm.cmd")
	if _, err := os.Stat(npmCmd); err == nil {
		return npmCmd
	}
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	return "npm"
}

func ensureFridaScripts(nodeExe string) (string, error) {
	dir := getFridaScriptsDir()
	os.MkdirAll(dir, 0755)

	files := []string{"frida-scripts/frida_inject.mjs", "frida-scripts/cloudHookAndroid.js", "frida-scripts/global-shim.js"}
	for _, f := range files {
		data, err := fridaScripts.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("读取嵌入文件 %s: %v", f, err)
		}
		dst := filepath.Join(dir, filepath.Base(f))
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return "", fmt.Errorf("写入 %s: %v", dst, err)
		}
	}

	fridaModule := filepath.Join(dir, "node_modules", "frida")
	bridgeModule := filepath.Join(dir, "node_modules", "frida-java-bridge")
	_, fridaErr := os.Stat(fridaModule)
	_, bridgeErr := os.Stat(bridgeModule)
	if os.IsNotExist(fridaErr) || os.IsNotExist(bridgeErr) {
		CallJsAlert("首次安装", "正在安装 Frida 依赖 (npm install)，首次约需1-2分钟...")
		pkgJson := `{"name":"sunny-frida","private":true,"type":"module"}`
		os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJson), 0644)

		npmPath := findNpm(nodeExe)
		cmd := exec.Command(npmPath, "install", "frida", "frida-java-bridge", "--no-optional")
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("npm install 失败: %s\n%v", string(out), err)
		}
	}

	return filepath.Join(dir, "frida_inject.mjs"), nil
}

func runCloudHook() {
	defer func() {
		cloudHook.mu.Lock()
		cloudHook.running = false
		cloudHook.cmd = nil
		cloudHook.mu.Unlock()
		CallJs("安卓云函数Hook状态", map[string]interface{}{"running": false})
	}()

	cloudHook.addLog("检测 ADB 设备...")
	device, err := CheckADBDevice()
	if err != nil {
		cloudHook.addLog("错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "ADB: "+err.Error())
		return
	}
	cloudHook.mu.Lock()
	cloudHook.device = device
	cloudHook.mu.Unlock()
	cloudHook.addLog("设备: " + device)

	cloudHook.addLog("确保 frida-server...")
	if err := EnsureFridaServer(); err != nil {
		cloudHook.addLog("错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "frida-server: "+err.Error())
		return
	}
	cloudHook.addLog("frida-server OK")

	nodeExe, err := ensureNodeJS()
	if err != nil {
		cloudHook.addLog("Node.js 错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "Node.js: "+err.Error())
		return
	}

	cloudHook.addLog("释放脚本...")
	injectScript, err := ensureFridaScripts(nodeExe)
	if err != nil {
		cloudHook.addLog("错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "脚本释放: "+err.Error())
		return
	}
	cloudHook.addLog("脚本就绪")

	cloudHook.addLog("启动 node 注入...")
	cmd := exec.Command(nodeExe, injectScript)
	cmd.Dir = filepath.Dir(injectScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cloudHook.addLog("管道失败: " + err.Error())
		CallJsAlert("安卓Hook失败", "管道: "+err.Error())
		return
	}
	stderrPipe, _ := cmd.StderrPipe()

	cloudHook.mu.Lock()
	cloudHook.cmd = cmd
	cloudHook.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cloudHook.addLog("node 启动失败: " + err.Error())
		CallJsAlert("安卓Hook失败", "node: "+err.Error())
		return
	}

	if stderrPipe != nil {
		go func() {
			sc := bufio.NewScanner(stderrPipe)
			for sc.Scan() {
				cloudHook.addLog("[stderr] " + sc.Text())
			}
		}()
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		if cloudHook.stopping {
			break
		}
		processCloudHookLine(scanner.Text())
	}

	cmd.Wait()
	cloudHook.addLog("Hook 已停止")
}

func processCloudHookLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	var fridaMsg FridaMessage
	if err := json.Unmarshal([]byte(line), &fridaMsg); err != nil {
		cloudHook.addLog("[stdout] " + line)
		return
	}

	payload := fridaMsg.Payload
	if fridaMsg.Type == "error" {
		cloudHook.addLog("脚本错误: " + payload)
		return
	}
	if payload == "" {
		return
	}

	var msg CloudHookMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return
	}

	switch msg.Type {
	case "cloud-request":
		handleCloudRequest(msg)
	case "cloud-response":
		handleCloudResponse(msg)
	case "hook-ready":
		cloudHook.addLog("Hook 就绪")
	case "log":
		cloudHook.addLog(msg.Message)
		if strings.Contains(msg.Message, "hook 完成") {
			CallJs("安卓云函数Hook状态", map[string]interface{}{"running": true})
			CallJsAlert("安卓Hook", msg.Message+"，开始监听云函数调用")
		}
	case "error":
		cloudHook.addLog("错误: " + msg.Message)
	}
}

func handleCloudRequest(msg CloudHookMessage) {
	theology := HashMap.CreateUniqueID()

	funcName := extractCloudFuncName(msg.Param2)
	cloudURL := "wx-cloud://" + msg.ApiType + "/" + funcName

	req := &MapHash.Request{
		PID:    "WeChat:Android",
		Method: "CLOUD",
		URL:    cloudURL,
		Body:   []byte(msg.Param2),
	}
	req.Display = true
	req.Header = http.Header{"X-Api-Type": {msg.ApiType}}
	req.SendTime = time.Now().Format("15:04:05.000")
	req.Response.StateCode = 200
	req.Response.Header = http.Header{"Content-Type": {"application/json"}}
	HashMap.SetRequest(theology, req)

	listInfo := &ListInfo{
		Theology: theology,
		State:    "→发送",
		URL:      cloudURL,
		HOST:     "wx-cloud",
		ClientIP: "Android",
		PID:      "WeChat",
		Method:   "CLOUD",
		Ico:      "上行",
		Type:     "application/json",
		Len:      strconv.Itoa(len(msg.Param2)),
		SendTime: time.Now().Format("15:04:05"),
	}

	AddInsertList(listInfo)
}

func handleCloudResponse(msg CloudHookMessage) {
	theology := HashMap.CreateUniqueID()

	respBody := msg.Param1
	if respBody == "" {
		respBody = msg.Param2
	}

	cloudURL := "wx-cloud://response/id=" + strconv.Itoa(msg.ID)

	req := &MapHash.Request{
		PID:    "WeChat:Android",
		Method: "CLOUD",
		URL:    cloudURL,
	}
	req.Display = true
	req.Header = http.Header{"X-Callback-ID": {strconv.Itoa(msg.ID)}}
	req.SendTime = time.Now().Format("15:04:05.000")
	req.Response.StateCode = 200
	req.Response.Body = []byte(respBody)
	req.Response.Header = http.Header{"Content-Type": {"application/json"}}
	req.RecTime = req.SendTime
	HashMap.SetRequest(theology, req)

	listInfo := &ListInfo{
		Theology: theology,
		State:    "←响应",
		URL:      cloudURL,
		HOST:     "wx-cloud",
		ClientIP: "Android",
		PID:      "WeChat",
		Method:   "CLOUD",
		Ico:      "下行",
		Type:     "application/json",
		Len:      strconv.Itoa(len(respBody)),
		SendTime: time.Now().Format("15:04:05"),
	}

	AddInsertList(listInfo)
}

func extractCloudFuncName(param2 string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(param2), &obj); err != nil {
		return "unknown"
	}
	if name, ok := obj["name"].(string); ok && name != "" {
		return name
	}
	if data, ok := obj["data"].(map[string]interface{}); ok {
		if name, ok := data["name"].(string); ok && name != "" {
			return name
		}
	}
	return "call"
}
