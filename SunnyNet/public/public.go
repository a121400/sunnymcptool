package public

import (
	srcpublic "github.com/qtgolang/SunnyNet/src/public"
	"sync/atomic"
)

const NULL = srcpublic.NULL
const RootCa = srcpublic.RootCa
const RootKey = srcpublic.RootKey

func AddTheology(n int) int64 {
	return atomic.AddInt64(&srcpublic.Theology, int64(n))
}

func SubString(str, left, right string) string {
	return srcpublic.SubString(str, left, right)
}

func WriteBytesToFile(data []byte, filename string) error {
	return srcpublic.WriteBytesToFile(data, filename)
}

func RemoveFile(filename string) error {
	return srcpublic.RemoveFile(filename)
}
