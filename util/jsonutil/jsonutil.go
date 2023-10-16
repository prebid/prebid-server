package jsonutil

import (
	"bytes"
	"encoding/json"
	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/prebid-server/errortypes"
	"io"
)

var comma = []byte(",")[0]
var colon = []byte(":")[0]
var sqBracket = []byte("]")[0]
var openCurlyBracket = []byte("{")[0]
var closingCurlyBracket = []byte("}")[0]
var quote = []byte(`"`)[0]

// Finds element in json byte array with any level of nesting
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

// Drops element from json byte array
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

// Unmarshal unmarshals a byte slice into the specified data structure without performing
// any validation on the data. An unmarshal error is returned if a non-validation error occurs.
func Unmarshal(data []byte, v interface{}) error {
	err := jsoniter.Config{
		EscapeHTML:             true,
		SortMapKeys:            true,
		ValidateJsonRawMessage: false,
	}.Froze().Unmarshal(data, v)

	if err != nil {
		return &errortypes.FailedToUnmarshal{
			Message: err.Error(),
		}
	}
	return nil
}

// UnmarshalValid validates and unmarshals a byte slice into the specified data structure
// returning an error if validation fails
func UnmarshalValid(data []byte, v interface{}) error {
	if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, v); err != nil {
		return &errortypes.FailedToUnmarshal{
			Message: err.Error(),
		}
	}
	return nil
}

// Marshal marshals a data structure into a byte slice without performing any validation
// on the data. A marshal error is returned if a non-validation error occurs.
func Marshal(v interface{}) ([]byte, error) {
	data, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(v)
	if err != nil {
		return nil, &errortypes.FailedToMarshal{
			Message: err.Error(),
		}
	}
	return data, nil
}
