//go:build windows && !cgo
// +build windows,!cgo

package Driver

type event struct {
	Go_threadStart       func()
	Go_threadEnd         func()
	Go_tcpConnectRequest func(id uint64, pConnInfo uintptr)
	Go_tcpConnected      func(id uint64, pConnInfo uintptr)
	Go_tcpClosed         func(id uint64, pConnInfo uintptr)
	Go_tcpReceive        func(id uint64, buf *byte, len int32)
	Go_tcpSend           func(id uint64, buf *byte, len int32)
	Go_tcpCanReceive     func(id uint64)
	Go_tcpCanSend        func(id uint64)
	Go_udpCreated        func(id uint64, pConnInfo uintptr)
	Go_udpConnectRequest func(id uint64, pConnReq uintptr)
	Go_udpClosed         func(id uint64, pConnInfo uintptr)
	Go_udpReceive        func(id uint64, remoteAddress uintptr, buf uintptr, length int32, options uintptr)
	Go_udpSend           func(id uint64, remoteAddress uintptr, buf uintptr, length int32, options uintptr)
	Go_udpCanReceive     func(id uint64)
	Go_udpCanSend        func(id uint64)
}

var Event = &event{}

func CgoDriverInit(driverName string, InitAddr uintptr) int32 {
	return -1
}
