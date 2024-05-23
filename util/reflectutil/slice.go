package reflectutil

import (
	"unsafe"

	"github.com/modern-go/reflect2"
)

// UnsafeSliceClone clones an existing slice using unsafe.Pointer conventions. Intended
// for use by json iterator extensions and should likely be used no where else. Nil
// behavior is undefined as checks are expected upstream.
func UnsafeSliceClone(ptr unsafe.Pointer, sliceType reflect2.SliceType) unsafe.Pointer {
	// it's also possible to use `sliceType.Elem().RType`, but that returns a `uintptr`
	// which causes `go vet` to emit a warning even though the usage is safe. this approach
	// of copying some internals from the reflect2 package avoids the cast of `uintptr` to
	// `unsafe.Pointer` which keeps `go vet` output clean.
	elemRType := unpackEFace(sliceType.Elem().Type1()).data

	header := (*sliceHeader)(ptr)
	newHeader := (*sliceHeader)(sliceType.UnsafeMakeSlice(header.Len, header.Cap))
	typedslicecopy(elemRType, *newHeader, *header)
	return unsafe.Pointer(newHeader)
}

// sliceHeader is copied from the reflect2 package v1.0.2.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// typedslicecopyis copied from the reflect2 package v1.0.2.
// it copies a slice of elemType values from src to dst,
// returning the number of elements copied.
//
//go:linkname typedslicecopy reflect.typedslicecopy
//go:noescape
func typedslicecopy(elemType unsafe.Pointer, dst, src sliceHeader) int

// eface is copied from the reflect2 package v1.0.2.
type eface struct {
	rtype unsafe.Pointer
	data  unsafe.Pointer
}

// unpackEFace is copied from the reflect2 package v1.0.2.
func unpackEFace(obj interface{}) *eface {
	return (*eface)(unsafe.Pointer(&obj))
}
