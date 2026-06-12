package gonix

import (
	"unsafe"

	raw "github.com/sund3RRR/nix-go-bindings"
)

func takeCString(ptr *byte) string {
	if ptr == nil {
		return ""
	}

	defer raw.StringFree(ptr)

	base := unsafe.Pointer(ptr)
	n := 0
	for *(*byte)(unsafe.Add(base, n)) != 0 {
		n++
	}

	return string(unsafe.Slice(ptr, n))
}
