//go:build !cgo
// +build !cgo

package public

func Free(id uintptr) {
}

func PointerPtr(data interface{}) uintptr {
	return 0
}
