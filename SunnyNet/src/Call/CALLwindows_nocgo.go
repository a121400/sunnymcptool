//go:build windows && !cgo
// +build windows,!cgo

package Call

func Call(address int, arg ...interface{}) int {
	return 0
}
