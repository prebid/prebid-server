package jsonutil

import (
	"encoding/json"
	"errors"
	"reflect"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/prebid/prebid-server/v3/errortypes"
)

// jsonConfigMergeClone uses the same configuration as the `ConfigCompatibleWithStandardLibrary` profile
// with extensions added to support the merge clone behavior.
var jsonConfigMergeClone = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
}.Froze()

func init() {
	jsonConfigMergeClone.RegisterExtension(&mergeCloneExtension{})
}

// MergeClone unmarshals json data on top of an existing object and clones pointers of
// the existing object before setting new values. Slices and maps are also cloned.
// Fields of type json.RawMessage are merged rather than replaced.
func MergeClone(v any, data json.RawMessage) error {
	err := jsonConfigMergeClone.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	return &errortypes.FailedToUnmarshal{
		Message: tryExtractErrorMessage(err),
	}
}

type mergeCloneExtension struct {
	jsoniter.DummyExtension
}

func (e *mergeCloneExtension) CreateDecoder(typ reflect2.Type) jsoniter.ValDecoder {
	if typ == jsonRawMessageType {
		return &extMergeDecoder{sliceType: typ.(*reflect2.UnsafeSliceType)}
	}
	return nil
}

func (e *mergeCloneExtension) DecorateDecoder(typ reflect2.Type, decoder jsoniter.ValDecoder) jsoniter.ValDecoder {
	if typ.Kind() == reflect.Ptr {
		ptrType := typ.(*reflect2.UnsafePtrType)
		return &ptrCloneDecoder{valueDecoder: decoder, elemType: ptrType.Elem()}
	}

	// don't use json.RawMessage on fields handled by extMergeDecoder
	if typ.Kind() == reflect.Slice && typ != jsonRawMessageType {
		return &sliceCloneDecoder{valueDecoder: decoder, sliceType: typ.(*reflect2.UnsafeSliceType)}
	}

	if typ.Kind() == reflect.Map {
		return &mapCloneDecoder{valueDecoder: decoder, mapType: typ.(*reflect2.UnsafeMapType)}
	}

	return decoder
}

type ptrCloneDecoder struct {
	elemType     reflect2.Type
	valueDecoder jsoniter.ValDecoder
}

func (d *ptrCloneDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// don't clone if field is being set to nil. checking for nil "consumes" the null
	// token, so must be handled in this decoder.
	if iter.ReadNil() {
		*((*unsafe.Pointer)(ptr)) = nil
		return
	}

	// clone if there is an existing object. creation of new objects is handled by the
	// original decoder.
	if *((*unsafe.Pointer)(ptr)) != nil {
		obj := d.elemType.UnsafeNew()
		d.elemType.UnsafeSet(obj, *((*unsafe.Pointer)(ptr)))
		*((*unsafe.Pointer)(ptr)) = obj
	}

	d.valueDecoder.Decode(ptr, iter)
}

type sliceCloneDecoder struct {
	sliceType    *reflect2.UnsafeSliceType
	valueDecoder jsoniter.ValDecoder
}

func (d *sliceCloneDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// clear the field. a new slice will be created by the original decoder if needed.
	d.sliceType.UnsafeSetNil(ptr)

	// checking for nil "consumes" the null token, so must be handled in this decoder.
	if iter.ReadNil() {
		return
	}

	d.valueDecoder.Decode(ptr, iter)
}

type mapCloneDecoder struct {
	mapType      *reflect2.UnsafeMapType
	valueDecoder jsoniter.ValDecoder
}

func (d *mapCloneDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// don't clone if field is being set to nil. checking for nil "consumes" the null
	// token, so must be handled in this decoder.
	if iter.ReadNil() {
		*(*unsafe.Pointer)(ptr) = nil
		d.mapType.UnsafeSet(ptr, d.mapType.UnsafeNew())
		return
	}

	// clone if there is an existing object. creation of new objects is handled by the
	// original decoder.
	if !d.mapType.UnsafeIsNil(ptr) {
		clone := d.mapType.UnsafeMakeMap(0)
		mapIter := d.mapType.UnsafeIterate(ptr)
		for mapIter.HasNext() {
			key, elem := mapIter.UnsafeNext()
			d.mapType.UnsafeSetIndex(clone, key, elem)
		}
		d.mapType.UnsafeSet(ptr, clone)
	}

	d.valueDecoder.Decode(ptr, iter)
}

type extMergeDecoder struct {
	sliceType *reflect2.UnsafeSliceType
}

func (d *extMergeDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// incoming nil value, keep existing
	if iter.ReadNil() {
		return
	}

	existing := *((*json.RawMessage)(ptr))
	incoming := iter.SkipAndReturnBytes()

	// check for read errors to avoid calling jsonpatch.MergePatch on bad data.
	if iter.Error != nil {
		return
	}

	// existing empty value, use incoming
	if len(existing) == 0 {
		*((*json.RawMessage)(ptr)) = incoming
		return
	}

	// non-empty incoming and existing values, merge
	merged, err := jsonpatch.MergePatch(existing, incoming)
	if err != nil {
		if errors.Is(err, jsonpatch.ErrBadJSONDoc) {
			iter.ReportError("merge", "invalid json on existing object")
		} else {
			iter.ReportError("merge", err.Error())
		}
		return
	}

	*((*json.RawMessage)(ptr)) = merged
}
