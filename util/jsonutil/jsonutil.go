package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"github.com/prebid/prebid-server/v3/errortypes"
)

const (
	comma               byte = ','
	colon               byte = ':'
	sqBracket           byte = ']'
	closingCurlyBracket byte = '}'
)

// RawMessage to allow replacing json with jsonutil
type RawMessage = json.RawMessage

// FindElement finds element in json byte array with any level of nesting
func FindElement(extension []byte, elementNames ...string) (bool, int64, int64, error) {
	elementName := elementNames[0]
	buf := bytes.NewBuffer(extension)
	dec := json.NewDecoder(buf)
	found := false
	var startIndex, endIndex int64
	var i interface{}
	for {
		token, err := dec.Token()
		if err == io.EOF {
			// io.EOF is a successful end
			break
		}
		if err != nil {
			return false, -1, -1, err
		}
		if token == elementName {
			err := dec.Decode(&i)
			if err != nil {
				return false, -1, -1, err
			}
			endIndex = dec.InputOffset()

			if dec.More() {
				//if there were other elements before
				if extension[startIndex] == comma {
					startIndex++
				}
				for {
					//structure has more elements, need to find index of comma
					if extension[endIndex] == comma {
						endIndex++
						break
					}
					endIndex++
				}
			}
			found = true
			break
		} else {
			startIndex = dec.InputOffset()
		}
	}
	if found {
		if len(elementNames) == 1 {
			return found, startIndex, endIndex, nil
		} else if len(elementNames) > 1 {
			for {
				//find the beginning of nested element
				if extension[startIndex] == colon {
					startIndex++
					break
				}
				startIndex++
			}
			for {
				if endIndex == int64(len(extension)) {
					endIndex--
				}

				//if structure had more elements, need to find index of comma at the end
				if extension[endIndex] == sqBracket || extension[endIndex] == closingCurlyBracket {
					break
				}

				if extension[endIndex] == comma {
					endIndex--
					break
				} else {
					endIndex--
				}
			}
			if found {
				found, startInd, endInd, err := FindElement(extension[startIndex:endIndex], elementNames[1:]...)
				return found, startIndex + startInd, startIndex + endInd, err
			}
			return found, startIndex, startIndex, nil
		}
	}
	return found, startIndex, endIndex, nil
}

// DropElement drops element from json byte array
// - Doesn't support drop element from json list
// - Keys in the path can skip levels
// - First found element will be removed
func DropElement(extension []byte, elementNames ...string) ([]byte, error) {
	found, startIndex, endIndex, err := FindElement(extension, elementNames...)
	if err != nil {
		return nil, err
	}
	if found {
		extension = append(extension[:startIndex], extension[endIndex:]...)
	}
	return extension, nil
}

// jsonConfigValidationOn attempts to maintain compatibility with the standard library which
// includes enabling validation
var jsonConfigValidationOn = jsoniter.ConfigCompatibleWithStandardLibrary

// jsonConfigValidationOff disables validation
var jsonConfigValidationOff = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: false,
}.Froze()

// Unmarshal unmarshals a byte slice into the specified data structure without performing
// any validation on the data. An unmarshal error is returned if a non-validation error occurs.
func Unmarshal(data []byte, v interface{}) error {
	err := jsonConfigValidationOff.Unmarshal(data, v)
	if err != nil {
		return &errortypes.FailedToUnmarshal{
			Message: tryExtractErrorMessage(err),
		}
	}
	return nil
}

// UnmarshalValid validates and unmarshals a byte slice into the specified data structure
// returning an error if validation fails
func UnmarshalValid(data []byte, v interface{}) error {
	if err := jsonConfigValidationOn.Unmarshal(data, v); err != nil {
		return &errortypes.FailedToUnmarshal{
			Message: tryExtractErrorMessage(err),
		}
	}
	return nil
}

// Marshal marshals a data structure into a byte slice without performing any validation
// on the data. A marshal error is returned if a non-validation error occurs.
func Marshal(v interface{}) ([]byte, error) {
	data, err := jsonConfigValidationOn.Marshal(v)
	if err != nil {
		return nil, &errortypes.FailedToMarshal{
			Message: err.Error(),
		}
	}
	return data, nil
}

// tryExtractErrorMessage attempts to extract a sane error message from the json-iter package. The errors
// returned from that library are not types and include a lot of extra information we don't want to respond with.
// This is hacky, but it's the only downside to the json-iter library.
func tryExtractErrorMessage(err error) string {
	msg := err.Error()

	msgEndIndex := strings.LastIndex(msg, ", error found in #")
	if msgEndIndex == -1 {
		return msg
	}

	msgStartIndex := strings.Index(msg, ": ")
	if msgStartIndex == -1 {
		return msg
	}

	operationStack := []string{msg[0:msgStartIndex]}
	for {
		msgStartIndexNext := strings.Index(msg[msgStartIndex+2:], ": ")

		// no more matches
		if msgStartIndexNext == -1 {
			break
		}

		// matches occur after the end message marker (sanity check)
		if (msgStartIndex + msgStartIndexNext) >= msgEndIndex {
			break
		}

		// match should not contain a space, indicates operation is really an error message
		match := msg[msgStartIndex+2 : msgStartIndex+2+msgStartIndexNext]
		if strings.Contains(match, " ") {
			break
		}

		operationStack = append(operationStack, match)
		msgStartIndex += msgStartIndexNext + 2
	}

	if len(operationStack) > 1 && isLikelyDetailedErrorMessage(msg[msgStartIndex+2:]) {
		return "cannot unmarshal " + operationStack[len(operationStack)-2] + ": " + msg[msgStartIndex+2:msgEndIndex]
	}

	return msg[msgStartIndex+2 : msgEndIndex]
}

// isLikelyDetailedErrorMessage checks if the json unmarshal error contains enough information such
// that the caller clearly understands the context, where the structure name is not needed.
func isLikelyDetailedErrorMessage(msg string) bool {
	return !strings.HasPrefix(msg, "request.")
}

// RawMessageExtension will call json.Compact() on every json.RawMessage field when getting marshalled.
type RawMessageExtension struct {
	jsoniter.DummyExtension
}

// CreateEncoder substitutes the default jsoniter encoder of the json.RawMessage type with ours, that
// calls json.Compact() before writting to the stream
func (e *RawMessageExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	if typ == jsonRawMessageType {
		return &rawMessageCodec{}
	}
	return nil
}

var jsonRawMessageType = reflect2.TypeOfPtr((*json.RawMessage)(nil)).Elem()

// rawMessageCodec implements jsoniter.ValEncoder interface so we can override the default json.RawMessage Encode()
// function with our implementation
type rawMessageCodec struct{}

func (codec *rawMessageCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	if ptr != nil {
		jsonRawMsg := *(*[]byte)(ptr)

		dst := bytes.NewBuffer(make([]byte, 0, len(jsonRawMsg)))
		if err := json.Compact(dst, jsonRawMsg); err == nil {
			stream.Write(dst.Bytes())
		}
	}
}

func (codec *rawMessageCodec) IsEmpty(ptr unsafe.Pointer) bool {
	return ptr == nil || len(*((*json.RawMessage)(ptr))) == 0
}
