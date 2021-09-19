package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"
)

var comma = []byte(",")[0]
var colon = []byte(":")[0]

func findElementIndexes(extension []byte, elementName string) (bool, int64, int64, error) {
	found := false
	buf := bytes.NewBuffer(extension)
	dec := json.NewDecoder(buf)
	var startIndex int64
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
			found = true
			endIndex := dec.InputOffset()

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
			return found, startIndex, endIndex, nil
		} else {
			startIndex = dec.InputOffset()
		}

	}

	return false, -1, -1, nil
}

func DropElement(extension []byte, elementName string) ([]byte, error) {
	found, startIndex, endIndex, err := findElementIndexes(extension, elementName)
	if found {
		extension = append(extension[:startIndex], extension[endIndex:]...)
	}
	return extension, err
}

func FindElement(extension []byte, elementName string) (bool, []byte, error) {

	found, startIndex, endIndex, err := findElementIndexes(extension, elementName)

	if found && err == nil {
		element := extension[startIndex:endIndex]
		index := 0
		for {
			if index < len(element) && element[index] != colon {
				index++
			} else {
				index++
				break
			}
		}
		element = element[index:]
		return found, element, err
	}

	return found, nil, err

}
