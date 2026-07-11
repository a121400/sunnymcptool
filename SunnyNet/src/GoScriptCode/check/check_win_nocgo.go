//go:build windows && !cgo
// +build windows,!cgo

package check

import "reflect"

func Check(Symbols map[string]map[string]reflect.Value) {
}
