// cloud_hook.go - 安卓云函数 Hook (Frida 注入)
//
// 为什么 SunnyNet 代理抓不到 MMTLS:
//   MMTLS 是微信自研加密协议(Mars框架)，非标准 TLS。
//   连接直接 TCP 到微信服务器(220.196.x.x:443)，绕过 HTTP 代理。
//   即使用 WinDivert 捕获 TCP，数据也是 MMTLS 加密的，无法 MitM 解密。
//
// 安卓方案: Frida 注入微信进程，Hook JNI 层 nativeInvokeHandler，
//   在 MMTLS 加密前拦截明文云函数请求/响应(wx-cloud://verifyPlugin/call 等)。
//
// PC/Windows 方案(未实现):
//   - wxaruntime/transfer API 主动调用已知 CGI (推荐，已在 wxhook 实现)
//     正确 CGI: /cgi-bin/mmbiz-bin/wxaapp/verifyplugin (非 js-verifyplugin)
//   - CE 内存扫描读取缓存的 host_sign/noncestr/timestamp
//   - Frida Windows 注入 WeChatAppEx.exe (需逆向 mmtls.cc)
//
// 参考: https://github.com/citizenlab/wechat-security-report
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
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

//go:embed frida-scripts/node_modules_win64.zip
var fridaNodeModulesZip []byte

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
	logLine := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	s.mu.Lock()
	s.logs = append(s.logs, logLine)
	if len(s.logs) > 200 {
		s.logs = s.logs[len(s.logs)-100:]
	}
	s.mu.Unlock()
	fmt.Println("[CloudHook]", msg)
	CallJs("安卓云函数Hook日志", map[string]interface{}{"log": logLine})
}

func CheckCloudHookEnv() map[string]interface{} {
	results := []map[string]interface{}{}

	cloudHook.addLog("=== 环境检查开始 ===")

	nodeExe, err := ensureNodeJS()
	if err != nil {
		cloudHook.addLog("✗ Node.js: " + err.Error())
		results = append(results, map[string]interface{}{"name": "Node.js", "ok": false, "msg": err.Error()})
	} else {
		cmd := hiddenCmd(nodeExe, "--version")
		ver, _ := cmd.Output()
		verStr := strings.TrimSpace(string(ver))
		cloudHook.addLog("✓ Node.js: " + verStr + " (" + nodeExe + ")")
		results = append(results, map[string]interface{}{"name": "Node.js", "ok": true, "msg": verStr})
	}

	adbExePath, err := ensureADB()
	if err != nil {
		cloudHook.addLog("✗ ADB: " + err.Error())
		results = append(results, map[string]interface{}{"name": "ADB", "ok": false, "msg": err.Error()})
	} else {
		adbPath = adbExePath
		cmd := hiddenCmd(adbExePath, "version")
		ver, _ := cmd.Output()
		verLine := strings.Split(strings.TrimSpace(string(ver)), "\n")
		verStr := "OK"
		if len(verLine) > 0 {
			verStr = strings.TrimSpace(verLine[0])
		}
		cloudHook.addLog("✓ ADB: " + verStr)
		results = append(results, map[string]interface{}{"name": "ADB", "ok": true, "msg": verStr})
	}

	_, err = ensureFridaScripts(nodeExe)
	if err != nil {
		cloudHook.addLog("✗ Frida 依赖: " + err.Error())
		results = append(results, map[string]interface{}{"name": "Frida 依赖", "ok": false, "msg": err.Error()})
	} else {
		fridaVer := GetFridaVersion()
		cloudHook.addLog("✓ Frida 依赖: v" + fridaVer)
		results = append(results, map[string]interface{}{"name": "Frida 依赖", "ok": true, "msg": "v" + fridaVer})
	}

	allOk := true
	for _, r := range results {
		if !r["ok"].(bool) {
			allOk = false
			break
		}
	}
	cloudHook.addLog(fmt.Sprintf("=== 环境检查完成: %v ===", allOk))

	return map[string]interface{}{
		"success": allOk,
		"results": results,
	}
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


const embeddedModulesVersion = "2"

func isFridaModuleValid(dir string) bool {
	markerPath := filepath.Join(dir, "node_modules", ".sunny_embedded_v")
	data, err := os.ReadFile(markerPath)
	if err != nil || strings.TrimSpace(string(data)) != embeddedModulesVersion {
		return false
	}
	fridaModule := filepath.Join(dir, "node_modules", "frida")
	bridgeModule := filepath.Join(dir, "node_modules", "frida-java-bridge")
	if _, err := os.Stat(fridaModule); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(bridgeModule); os.IsNotExist(err) {
		return false
	}
	binding := filepath.Join(fridaModule, "build", "frida_binding.node")
	if _, err := os.Stat(binding); os.IsNotExist(err) {
		return false
	}
	return true
}

func extractEmbeddedNodeModules(dir string) error {
	cloudHook.addLog("解压内置 frida 依赖...")
	CallJsAlert("首次安装", "正在解压 Frida 依赖，请稍候...")

	r, err := zip.NewReader(bytes.NewReader(fridaNodeModulesZip), int64(len(fridaNodeModulesZip)))
	if err != nil {
		return fmt.Errorf("读取内置 zip: %v", err)
	}

	fileCount := 0
	for _, f := range r.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		target := filepath.Join(dir, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("创建目录 %s: %v", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("创建父目录 %s: %v", filepath.Dir(target), err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("打开 %s: %v", name, err)
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return fmt.Errorf("创建文件 %s: %v", target, err)
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("写入 %s: %v", target, err)
		}
		fileCount++
	}

	markerPath := filepath.Join(dir, "node_modules", ".sunny_embedded_v")
	os.WriteFile(markerPath, []byte(embeddedModulesVersion), 0644)
	cloudHook.addLog(fmt.Sprintf("解压完成: %d 个文件", fileCount))
	return nil
}

func ensureFridaScripts(nodeExe string) (string, error) {
	dir := getFridaScriptsDir()
	os.MkdirAll(dir, 0755)
	cloudHook.addLog("脚本目录: " + dir)

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
		cloudHook.addLog("  释放: " + filepath.Base(f))
	}

	pkgJson := `{"name":"sunny-frida","private":true,"type":"module"}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJson), 0644)

	if isFridaModuleValid(dir) {
		cloudHook.addLog("Frida node_modules 已存在且有效")
		return filepath.Join(dir, "frida_inject.mjs"), nil
	}

	cloudHook.addLog("Frida node_modules 需要重新解压 (版本标记不匹配)...")
	os.RemoveAll(filepath.Join(dir, "node_modules"))
	os.Remove(filepath.Join(dir, "package-lock.json"))

	if err := extractEmbeddedNodeModules(dir); err != nil {
		return "", fmt.Errorf("解压 frida 依赖失败: %v", err)
	}

	if !isFridaModuleValid(dir) {
		return "", fmt.Errorf("解压后 frida 依赖校验失败")
	}

	cloudHook.addLog("Frida 依赖就绪")
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

	cloudHook.addLog("=== 启动安卓云函数 Hook ===")

	cloudHook.addLog("[1/5] 检查 Node.js...")
	nodeExe, err := ensureNodeJS()
	if err != nil {
		cloudHook.addLog("✗ Node.js 错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "Node.js: "+err.Error())
		return
	}
	cloudHook.addLog("✓ Node.js: " + nodeExe)

	cloudHook.addLog("[2/5] 检查 ADB...")
	adbExePath, err := ensureADB()
	if err != nil {
		cloudHook.addLog("✗ ADB 错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "ADB: "+err.Error())
		return
	}
	adbPath = adbExePath
	cloudHook.addLog("✓ ADB: " + adbExePath)

	cloudHook.addLog("[3/5] 检测设备...")
	device, err := CheckADBDevice()
	if err != nil {
		cloudHook.addLog("✗ 设备错误: " + err.Error())
		CallJsAlert("安卓Hook失败", err.Error())
		return
	}
	cloudHook.mu.Lock()
	cloudHook.device = device
	cloudHook.mu.Unlock()
	cloudHook.addLog("✓ 设备: " + device)

	cloudHook.addLog("[4/5] 释放 Frida 脚本...")
	injectScript, err := ensureFridaScripts(nodeExe)
	if err != nil {
		cloudHook.addLog("✗ 脚本释放错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "脚本释放: "+err.Error())
		return
	}
	cloudHook.addLog("✓ 脚本就绪: " + injectScript)

	cloudHook.addLog("[5/5] 确保 frida-server...")
	if err := EnsureFridaServer(); err != nil {
		cloudHook.addLog("✗ frida-server 错误: " + err.Error())
		CallJsAlert("安卓Hook失败", "frida-server: "+err.Error())
		return
	}
	cloudHook.addLog("✓ frida-server OK")

	cloudHook.addLog("=== 环境检查全部通过，启动注入 ===")
	cloudHook.addLog("执行: " + nodeExe + " " + injectScript)
	cmd := exec.Command(nodeExe, injectScript)
	cmd.Dir = filepath.Dir(injectScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cloudHook.addLog("✗ 管道创建失败: " + err.Error())
		CallJsAlert("安卓Hook失败", "管道: "+err.Error())
		return
	}
	stderrPipe, _ := cmd.StderrPipe()

	cloudHook.mu.Lock()
	cloudHook.cmd = cmd
	cloudHook.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cloudHook.addLog("✗ node 进程启动失败: " + err.Error())
		CallJsAlert("安卓Hook失败", "node: "+err.Error())
		return
	}
	cloudHook.addLog(fmt.Sprintf("node 进程已启动 (PID: %d)", cmd.Process.Pid))

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

	lineCount := 0
	for scanner.Scan() {
		if cloudHook.stopping {
			break
		}
		lineCount++
		processCloudHookLine(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		cloudHook.addLog("stdout 读取错误: " + err.Error())
	}

	exitErr := cmd.Wait()
	if exitErr != nil {
		cloudHook.addLog(fmt.Sprintf("node 进程退出: %v (共收到 %d 行输出)", exitErr, lineCount))
	} else {
		cloudHook.addLog(fmt.Sprintf("node 进程正常退出 (共收到 %d 行输出)", lineCount))
	}
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

