package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed mmtls_phone.py
var mmtlsPhoneScript string

var phoneSyncOnce sync.Once

func startMMTLSPhoneSync() {
	phoneSyncOnce.Do(func() {
		go phoneSyncLoop()
	})
}

func phoneSyncLoop() {
	fridaReady := false
	for {
		ok, needRetry := runPhoneSyncOnce(fridaReady)
		if ok {
			fridaReady = true
			// 脚本自己退出了（设备断开/微信重启），稍后再拉起。
			time.Sleep(2 * time.Second)
			continue
		}
		if !needRetry {
			// frida 没装之类的硬错误，别每 2 秒重来一次。
			return
		}
		time.Sleep(2 * time.Second)
	}
}

// runPhoneSyncOnce 启动一次 phone sync 子进程。
// 返回 (是否正常跑过, 是否值得重试)。
func runPhoneSyncOnce(fridaReady bool) (bool, bool) {
	dir := filepath.Dir(mmtlsKeyPath())
	_ = os.MkdirAll(dir, 0755)

	python := findPython()
	if python == "" {
		appendMMTLSPhoneLog(dir, "未找到 python，phone sync 跳过")
		return false, false
	}

	// 先确认 frida 绑定可用，避免拉起一个必然 ImportError 的进程再无限重启。
	if !fridaReady && !pythonHasFrida(python) {
		appendMMTLSPhoneLog(dir, "python 缺少 frida 模块，phone sync 跳过；pip install frida 后重启程序即可")
		return false, false
	}

	logFile, err := os.OpenFile(filepath.Join(dir, "mmtls_phone.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false, true
	}
	defer logFile.Close()

	// 走 stdin 传脚本：不再往磁盘写 .py，逻辑完全内置在 exe 里。
	// HiddenWindow + pythonw 双保险，GUI 启动不会再弹控制台。
	cmd := hiddenCmd(python, "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(mmtlsPhoneScript)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		appendMMTLSPhoneLog(dir, "启动 phone sync 失败: "+err.Error())
		return false, true
	}
	_ = os.WriteFile(filepath.Join(dir, "mmtls_phone.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	_ = cmd.Wait()
	_ = os.Remove(filepath.Join(dir, "mmtls_phone.pid"))
	// 进程退出通常意味着设备掉线/微信被杀，值得重试。
	return true, true
}

func appendMMTLSPhoneLog(dir, line string) {
	f, err := os.OpenFile(filepath.Join(dir, "mmtls_phone.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(time.Now().Format("2006-01-02 15:04:05") + " " + line + "\n")
}

// findPython 优先选 pythonw.exe：Windows 上它本身不带控制台窗口，
// 即使 HideWindow 失效也不会闪黑框。
func findPython() string {
	candidates := []string{
		`E:\python3.12\pythonw.exe`,
		`E:\python3.12\python.exe`,
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	for _, name := range []string{"pythonw", "python"} {
		if lp, err := exec.LookPath(name); err == nil {
			return lp
		}
	}
	return ""
}

// pythonHasFrida 用隐藏进程做一次 import 探测，不会弹控制台。
func pythonHasFrida(python string) bool {
	cmd := hiddenCmd(python, "-c", "import frida")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func procAlive(pidPath string) bool {
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Windows 上 FindProcess 不会检查进程是否还在，发一个 0 信号也不适用。
	// 用 tasklist 确认。
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		_ = proc.Release()
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}
