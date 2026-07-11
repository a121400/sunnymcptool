//go:build windows && !cgo
// +build windows,!cgo

package NFapi

func ApiInit() bool {
	return false
}
