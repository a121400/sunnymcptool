//go:build windows && !cgo
// +build windows,!cgo

package Proxifier

import "net"

func IsInit() bool {
	return false
}

func SetHandle(Handle func(conn net.Conn)) bool {
	return false
}
