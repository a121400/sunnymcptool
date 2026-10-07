//go:build windows
// +build windows

package Proxifier

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/qtgolang/SunnyNet/src/ProcessDrv/Info"
	"github.com/qtgolang/SunnyNet/src/ProcessDrv/ProcessCheck"
	"golang.org/x/sys/windows"
)

const (
	pipeBufferSize = 0x534
	pipeName       = `\\.\pipe\proxifier`
	mutexStd       = `Global\ProxifierStd300Mutex`
	mutexRun       = `Global\Proxifier32Mutex1040`
)

var HandleClientConn func(net.Conn)
var myPid = os.Getpid()

var (
	mu              sync.Mutex
	hMStop          atomic.Uintptr
	hMutexProxifier windows.Handle
	pipeSA          *windows.SecurityAttributes
)

func init() {
	sd, err := windows.NewSecurityDescriptor()
	if err == nil {
		_ = sd.SetDACL(nil, true, false)
		pipeSA = &windows.SecurityAttributes{
			Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
			SecurityDescriptor: sd,
		}
	}
	go proxifierLoop()
}

func proxifierLoop() {
	for {
		proxifierCreateMutex()
		if windows.Handle(hMStop.Load()) == 0 {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		waitPipe()
	}
}

func waitPipe() {
	name, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		time.Sleep(200 * time.Millisecond)
		return
	}
	hPipe, err := windows.CreateNamedPipe(
		name,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE,
		1, 0, 0, 0,
		pipeSA,
	)
	if err != nil || hPipe == 0 || hPipe == windows.InvalidHandle {
		time.Sleep(200 * time.Millisecond)
		return
	}
	err = windows.ConnectNamedPipe(hPipe, nil)
	if err != nil && err != windows.ERROR_PIPE_CONNECTED {
		_ = windows.CloseHandle(hPipe)
		return
	}
	buffer := make([]byte, pipeBufferSize)
	var n uint32
	if err = windows.ReadFile(hPipe, buffer, &n, nil); err == nil && n >= 4 {
		if binary.LittleEndian.Uint32(buffer[:4]) == n {
			handlePipe(hPipe, buffer)
		}
	}
	_ = windows.CloseHandle(hPipe)
}

func writePipe(hPipe windows.Handle, bs []byte) {
	if len(bs) < 1 {
		return
	}
	var written uint32
	_ = windows.WriteFile(hPipe, bs, &written, nil)
}

func proxifierCreateMutex() {
	name, err := windows.UTF16PtrFromString(mutexStd)
	if err != nil {
		return
	}
	h, err := windows.CreateMutex(nil, false, name)
	if h == 0 {
		return
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.ReleaseMutex(h)
		_ = windows.CloseHandle(h)
		return
	}
}

func startProxifier() int {
	proxifierCreateMutex()
	if windows.Handle(hMStop.Load()) != 0 {
		return 1
	}
	if hMutexProxifier != 0 {
		hMStop.Store(uintptr(hMutexProxifier))
		return 1
	}
	name, err := windows.UTF16PtrFromString(mutexRun)
	if err != nil {
		return 0
	}
	h, err := windows.CreateMutex(nil, false, name)
	if h == 0 || err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			_ = windows.CloseHandle(h)
		}
		hMutexProxifier = 0
		hMStop.Store(0)
		return 0
	}
	hMutexProxifier = h
	hMStop.Store(uintptr(h))
	return 1
}

func stopProxifier() int {
	if windows.Handle(hMStop.Load()) != 0 {
		hMStop.Store(0)
		return 1
	}
	return 0
}

func proxifierIsInit() bool {
	proxifierCreateMutex()
	name, err := windows.UTF16PtrFromString(mutexRun)
	if err != nil {
		return false
	}
	h, err := windows.OpenMutex(windows.MUTEX_ALL_ACCESS, false, name)
	if err != nil || h == 0 {
		return false
	}
	_ = windows.ReleaseMutex(h)
	_ = windows.CloseHandle(h)
	return true
}

func handlePipe(hPipe windows.Handle, raw []byte) {
	if len(raw) < 0x4EC+2 {
		return
	}
	__pid := int(binary.LittleEndian.Uint16(raw[0x4EC : 0x4EC+2]))
	path := utf16At(raw, 8)

	mu.Lock()
	Handle := HandleClientConn
	if __pid == myPid {
		mu.Unlock()
		return
	}
	if Handle == nil {
		mu.Unlock()
		return
	}
	mu.Unlock()
	fileName := filepath.Base(path)
	if ProcessCheck.CheckPidByName(int32(__pid), fileName) {
		return
	}
	if len(raw) < 0x419+8 {
		return
	}
	family := int16(binary.LittleEndian.Uint16(raw[0x419 : 0x419+2]))
	if family == 0 {
		WriteData := make([]byte, 1020)
		WriteData[0] = 0xfc
		WriteData[1] = 0x3
		WriteData[4] = 0x1
		WriteData[0x3f8] = 0x1
		writePipe(hPipe, WriteData)
		return
	}
	if family != 2 && family != 23 {
		return
	}
	domain := utf16At(raw, 528)
	port := int(binary.BigEndian.Uint16(raw[0x419+2 : 0x419+4]))
	if domain == "" {
		var bs []byte
		if family == 23 {
			if len(raw) < 0x419+8+16 {
				return
			}
			bs = append([]byte(nil), raw[0x419+8:0x419+8+16]...)
		} else {
			bs = append([]byte(nil), raw[0x419+4:0x419+8]...)
		}
		ip := net.IP(bs)
		domain = ip.String()
	}
	if port < 1 || port > 65535 {
		return
	}
	if Info.IsFilterRequests(fileName, domain) {
		return
	}
	var listener net.Listener
	var err error
	WriteData := make([]byte, 1020)
	WriteData[0] = 0xfc
	WriteData[1] = 0x03
	WriteData[9] = 0x00
	ISV6 := family == 23
	if ISV6 {
		WriteData[8] = 0x17
		listener, err = net.Listen("tcp", "[::1]:")
	} else {
		WriteData[8] = 0x02
		listener, err = net.Listen("tcp", "127.0.0.1:")
	}
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		connChan := make(chan net.Conn, 1)
		go func() {
			defer func() { _ = recover() }()
			conn, _ := listener.Accept()
			_ = listener.Close()
			connChan <- conn
		}()

		select {
		case conn := <-connChan:
			if conn != nil {
				ip := net.ParseIP(domain)
				if ip == nil {
					ip = net.ParseIP("[" + domain + "]")
				}
				_ISV6 := false
				if ip != nil && ip.To4() == nil && ip.To16() != nil {
					_ISV6 = true
				}
				obj := &proxyProcessInfo{listener: listener, RemoteAddress: domain, RemotePort: uint16(port), V6: _ISV6, Pid: fmt.Sprintf("%d", __pid)}
				connLocalAddr := conn.RemoteAddr().(*net.TCPAddr)
				connPort := uint16(connLocalAddr.Port)
				ProcessCheck.AddDevObj(connPort, obj)
				_ = conn.SetDeadline(time.Time{})
				Handle(conn)
				_ = conn.Close()
				ProcessCheck.DelDevObj(connPort)
			}
			_ = listener.Close()
			return
		case <-ctx.Done():
			_ = listener.Close()
			return
		}
	}()
	binary.BigEndian.PutUint16(WriteData[10:], uint16(listener.Addr().(*net.TCPAddr).Port))
	if ISV6 {
		WriteData[0x1f] = 0x01
		WriteData[0x3f0] = 0x17
	} else {
		WriteData[12] = 0x7f
		WriteData[13] = 0x00
		WriteData[14] = 0x00
		WriteData[15] = 0x01
		WriteData[0x3f0] = 0x02
	}
	WriteData[1012] = 0x06
	WriteData[1016] = 0x02
	writePipe(hPipe, WriteData)
}

type proxyProcessInfo struct {
	Id            uint64
	Pid           string
	RemoteAddress string
	RemotePort    uint16
	V6            bool
	listener      net.Listener
}

func (p *proxyProcessInfo) GetRemoteAddress() string {
	return p.RemoteAddress
}

func (p *proxyProcessInfo) GetRemotePort() uint16 {
	return p.RemotePort
}

func (p *proxyProcessInfo) GetPid() string {
	return p.Pid
}

func (p *proxyProcessInfo) IsV6() bool {
	return p.V6
}

func (p *proxyProcessInfo) ID() uint64 {
	return p.Id
}

func (p *proxyProcessInfo) Close() error {
	mu.Lock()
	if p.listener != nil {
		_ = p.listener.Close()
	}
	p.listener = nil
	mu.Unlock()
	return nil
}

func utf16At(buf []byte, off int) string {
	if off < 0 || off >= len(buf) {
		return ""
	}
	u := make([]uint16, 0, 32)
	for i := off; i+1 < len(buf); i += 2 {
		c := binary.LittleEndian.Uint16(buf[i:])
		if c == 0 {
			break
		}
		u = append(u, c)
	}
	return string(utf16.Decode(u))
}

func IsInit() bool {
	return proxifierIsInit() || HandleClientConn != nil
}

func SetHandle(Handle func(conn net.Conn)) bool {
	mu.Lock()
	var res int
	if Handle == nil {
		res = stopProxifier()
	} else {
		res = startProxifier()
	}
	HandleClientConn = Handle
	mu.Unlock()
	return res == 1
}
