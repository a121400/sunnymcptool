//go:build windows && !cgo
// +build windows,!cgo

package WinDivert

import (
	"errors"
)

func Open(filter string, layer Layer, priority int16, flags uint64) (h *Handle, err error) {
	return nil, errors.New("WinDivert requires CGO (C compiler)")
}

func CalcChecksums(buffer []byte, address *Address, flags uint64) bool {
	return false
}
