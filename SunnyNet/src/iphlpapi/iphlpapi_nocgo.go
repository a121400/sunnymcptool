//go:build windows && !cgo
// +build windows,!cgo

package iphlpapi

func init() {
	R1()
}

func CloseCurrentSocket(PID int, ulAf uint) {
}

func IsPortListening(port int) bool {
	return false
}
