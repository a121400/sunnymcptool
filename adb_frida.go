package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/ulikunitz/xz"
)

const FridaServerPath = "/data/local/tmp/frida-server"

func getAdbDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.TempDir()
	}
	return filepath.Join(appData, "SunnyNet", "adb")
}

func getAdbExe() string {
	return filepath.Join(getAdbDir(), "platform-tools", "adb.exe")
}

func findAdb() string {
	builtinAdb := getAdbExe()
	if _, err := os.Stat(builtinAdb); err == nil {
		return builtinAdb
	}
	if path, err := exec.LookPath("adb"); err == nil {
		return path
	}
	return ""
}

func ensureADB() (string, error) {
	if adbPath := findAdb(); adbPath != "" {
		cloudHook.addLog("ADB 已找到: " + adbPath)
		return adbPath, nil
	}

	cloudHook.addLog("ADB 未找到，开始下载 Android Platform Tools...")
	CallJsAlert("环境安装", "正在下载 ADB 工具，请稍候...")

	dir := getAdbDir()
	os.MkdirAll(dir, 0755)

	url := "https://dl.google.com/android/repository/platform-tools-latest-windows.zip"
	zipPath := filepath.Join(dir, "platform-tools.zip")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载 ADB 失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载 ADB 返回 HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(zipPath)
		return "", fmt.Errorf("下载 ADB 写入失败: %v", err)
	}
	cloudHook.addLog("ADB 下载完成，正在解压...")

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		os.Remove(zipPath)
		return "", fmt.Errorf("打开 ADB zip: %v", err)
	}
	for _, zf := range r.File {
		target := filepath.Join(dir, filepath.FromSlash(zf.Name))
		if zf.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, _ := zf.Open()
		out, _ := os.Create(target)
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	r.Close()
	os.Remove(zipPath)

	adbExePath := getAdbExe()
	if _, err := os.Stat(adbExePath); err != nil {
		return "", fmt.Errorf("解压后找不到 adb.exe")
	}
	cloudHook.addLog("ADB 安装完成: " + adbExePath)
	return adbExePath, nil
}

var adbPath string

func hiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func GetFridaVersion() string {
	scriptsDir := getFridaScriptsDir()
	pkgJson := filepath.Join(scriptsDir, "node_modules", "frida", "package.json")
	data, err := os.ReadFile(pkgJson)
	if err == nil {
		re := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
		if m := re.FindSubmatch(data); len(m) > 1 {
			return string(m[1])
		}
	}
	out, err := hiddenCmd("frida", "--version").Output()
	if err != nil {
		return "17.9.11"
	}
	return strings.TrimSpace(string(out))
}

func adbExec(args ...string) (string, error) {
	exe := adbPath
	if exe == "" {
		exe = findAdb()
	}
	if exe == "" {
		return "", fmt.Errorf("ADB 未安装")
	}
	cmd := hiddenCmd(exe, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func adbShellSu(command string) (string, error) {
	return adbExec("shell", fmt.Sprintf("su -c %s", command))
}

func CheckADBDevice() (string, error) {
	out, err := adbExec("devices")
	if err != nil {
		return "", fmt.Errorf("ADB 未安装或未配置到 PATH")
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("未检测到已连接的 Android 设备")
}

func GetDeviceArch() string {
	out, err := adbExec("shell", "getprop", "ro.product.cpu.abi")
	if err != nil {
		return "arm64"
	}
	abi := strings.TrimSpace(out)
	switch {
	case strings.Contains(abi, "arm64"), strings.Contains(abi, "aarch64"):
		return "arm64"
	case strings.Contains(abi, "armeabi"), strings.Contains(abi, "arm"):
		return "arm"
	case strings.Contains(abi, "x86_64"):
		return "x86_64"
	case strings.Contains(abi, "x86"):
		return "x86"
	default:
		return "arm64"
	}
}

func IsFridaServerRunning() bool {
	out, _ := adbShellSu("ps -A")
	return regexp.MustCompile(`root.*S\s+frida-server`).MatchString(out)
}

func StartFridaServer() error {
	adbShellSu("setenforce 0")
	adbShellSu("killall frida-server")
	time.Sleep(500 * time.Millisecond)

	cmd := hiddenCmd("adb", "shell", fmt.Sprintf("su -c %s -D", FridaServerPath))
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frida-server 失败: %v", err)
	}
	go func() { cmd.Wait() }()

	time.Sleep(2 * time.Second)
	if IsFridaServerRunning() {
		return nil
	}
	return fmt.Errorf("frida-server 启动后未检测到运行状态")
}

func IsFridaServerInstalled() bool {
	out, _ := adbShellSu(fmt.Sprintf("ls %s", FridaServerPath))
	return strings.Contains(out, "frida-server")
}

func InstallFridaServer() error {
	fridaVersion := GetFridaVersion()
	arch := GetDeviceArch()
	fileName := fmt.Sprintf("frida-server-%s-android-%s.xz", fridaVersion, arch)
	url := fmt.Sprintf("https://github.com/frida/frida/releases/download/%s/%s", fridaVersion, fileName)

	tmpDir := os.TempDir()
	xzPath := filepath.Join(tmpDir, fileName)
	binPath := filepath.Join(tmpDir, fmt.Sprintf("frida-server-%s-android-%s", fridaVersion, arch))

	cloudHook.addLog(fmt.Sprintf("下载 frida-server v%s (%s)...", fridaVersion, arch))
	CallJsAlert("安装 frida-server", fmt.Sprintf("正在下载 frida-server v%s，请稍候...", fridaVersion))
	if err := downloadFile(url, xzPath); err != nil {
		return fmt.Errorf("下载 frida-server 失败: %v", err)
	}
	cloudHook.addLog("frida-server 下载完成，解压中...")

	if err := decompressXZ(xzPath, binPath); err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}

	cloudHook.addLog("推送 frida-server 到设备...")
	if _, err := adbExec("push", binPath, FridaServerPath); err != nil {
		return fmt.Errorf("推送失败: %v", err)
	}
	adbShellSu(fmt.Sprintf("chmod 755 %s", FridaServerPath))

	os.Remove(xzPath)
	os.Remove(binPath)
	cloudHook.addLog("frida-server 安装完成")
	return nil
}

func decompressXZ(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := xz.NewReader(f)
	if err != nil {
		return fmt.Errorf("xz reader: %v", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	return err
}

func downloadFile(url, dst string) error {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 302 || resp.StatusCode == 301 {
		loc := resp.Header.Get("Location")
		if loc != "" {
			return downloadFile(loc, dst)
		}
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

type WeChatProcess struct {
	PID  int
	Name string
}

func FindWeChatProcesses() ([]WeChatProcess, error) {
	out, err := adbShellSu("ps -A")
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %v", err)
	}

	var procs []WeChatProcess
	re := regexp.MustCompile(`\S+\s+(\d+)\s+\d+\s+\d+\s+\d+\s+\S+\s+\S+\s+\S+\s+(com\.tencent\.mm\S*)`)
	for _, match := range re.FindAllStringSubmatch(out, -1) {
		pid := 0
		fmt.Sscanf(match[1], "%d", &pid)
		if pid > 0 {
			procs = append(procs, WeChatProcess{PID: pid, Name: match[2]})
		}
	}
	return procs, nil
}

func FindAppbrandProcess() (int, error) {
	procs, err := FindWeChatProcesses()
	if err != nil {
		return 0, err
	}
	for _, p := range procs {
		if strings.Contains(p.Name, "appbrand") {
			return p.PID, nil
		}
	}
	if len(procs) > 0 {
		return procs[0].PID, nil
	}
	return 0, fmt.Errorf("未找到微信进程")
}

func GetDeviceFridaVersion() string {
	out, _ := adbShellSu(fmt.Sprintf("%s --version", FridaServerPath))
	return strings.TrimSpace(out)
}

func EnsureFridaServer() error {
	cliVersion := GetFridaVersion()
	cloudHook.addLog("frida CLI 版本: " + cliVersion)

	if IsFridaServerInstalled() {
		deviceVersion := GetDeviceFridaVersion()
		cloudHook.addLog("设备 frida-server 版本: " + deviceVersion)
		if deviceVersion != cliVersion {
			cloudHook.addLog(fmt.Sprintf("版本不匹配 (CLI=%s, Server=%s)，重新安装...", cliVersion, deviceVersion))
			adbShellSu("killall frida-server")
			time.Sleep(500 * time.Millisecond)
			if err := InstallFridaServer(); err != nil {
				return err
			}
		} else {
			cloudHook.addLog("frida-server 版本匹配")
		}
	} else {
		cloudHook.addLog("设备未安装 frida-server，开始安装...")
		if err := InstallFridaServer(); err != nil {
			return err
		}
	}

	if IsFridaServerRunning() {
		cloudHook.addLog("frida-server 已在运行")
		return nil
	}
	cloudHook.addLog("启动 frida-server...")
	return StartFridaServer()
}
