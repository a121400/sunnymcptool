//go:build windows && !cgo
// +build windows,!cgo

package Info

import (
	"bufio"
	"github.com/qtgolang/SunnyNet/src/public"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

func GetSystemDirectory() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetSystemDirectoryW")
	buf := make([]uint16, 260)
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:r])
}

func Wow64DisableWow64FsRedirection() uintptr {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("Wow64DisableWow64FsRedirection")
	var oldValue uintptr
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&oldValue)))
	if r == 0 {
		return 0
	}
	return oldValue
}

func Wow64RevertWow64FsRedirection(oldValue uintptr) bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("Wow64RevertWow64FsRedirection")
	r, _, _ := proc.Call(oldValue)
	return r != 0
}

var (
	WindowsDirectory = GetWindowsDirectory()
)

const WindowsX64 = 4<<(^uintptr(0)>>63) == 8

var Is64Windows = IsX64CPU()

func IsX64CPU() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	GetSystemWow64DirectoryA := kernel32.NewProc("GetSystemWow64DirectoryA")
	Lstrcpyn := kernel32.NewProc("lstrcpyn")
	lpBuffer := make([]byte, 255)
	p := uintptr(unsafe.Pointer(&lpBuffer[0]))
	r, _, _ := Lstrcpyn.Call(p, p, 0)
	r, _, _ = GetSystemWow64DirectoryA.Call(r, 255)
	return r > 0
}

func GetWindowsDirectory() string {
	winDir := os.Getenv("windir")
	if winDir == "" {
		winDir = os.Getenv("SystemRoot")
	}
	if winDir == "" {
		return ""
	}
	if winDir[len(winDir)-1:] != "\\" {
		winDir += "\\"
	}
	return winDir
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func WriteFile(path string, data []byte) {
	if checkFileIsExist(path) {
		err := os.Remove(path)
		if err != nil {
			return
		}
	}
	f, err1 := os.Create(path)
	if err1 == nil {
		_, err1 = f.Write(data)
		if err1 != nil {
			return
		}
		err1 = f.Close()
		if err1 != nil {
			return
		}
	}
}

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func ExecCommand(commandName string, params []string) string {
	cmd := exec.Command(commandName, params...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err.Error()
	}
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	_ = cmd.Start()
	var s []byte
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadBytes('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		s = public.BytesCombine(s, line)
	}
	return string(s)
}

func IsFilterRequests(fileName, addr string) bool {
	if strings.Index(strings.ToLower(fileName), "wechat.exe") != -1 && (strings.Contains(addr, "::1") || strings.Contains(addr, "127.0.0.1")) {
		return true
	}
	if strings.Index(strings.ToLower(fileName), "steamwebhelper.exe") != -1 && (strings.Contains(addr, "::1") || strings.Contains(addr, "127.0.0.1")) {
		return true
	}
	return false
}

type DrvInfo interface {
	GetRemoteAddress() string
	GetRemotePort() uint16
	GetPid() string
	IsV6() bool
	ID() uint64
	Close()
}

var Name = make(map[string]bool)
var Pid = make(map[uint32]bool)
var Proxy = make(map[uint16]DrvInfo)
var Lock sync.Mutex

var HookProcess bool

func HookAllProcess(open, StopNetwork bool) {
}

func GetTcpConnectInfo(u uint16) DrvInfo {
	return nil
}

func DelTcpConnectInfo(u uint16) {
}

func AddName(u string) bool {
	return true
}

func DelName(u string) bool {
	return true
}

func AddPid(u uint32) bool {
	return true
}

func DelPid(u uint32) bool {
	return true
}

func CancelAll() bool {
	return true
}
