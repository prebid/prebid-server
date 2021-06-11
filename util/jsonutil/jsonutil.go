package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"
)

var comma = []byte(",")[0]

func DropElement(extension []byte, elementName string) ([]byte, error) {
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
			return nil, err
		}

		if token == elementName {
			err := dec.Decode(&i)
			if err != nil {
				return nil, err
			}
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

			extension = append(extension[:startIndex], extension[endIndex:]...)
			break
		} else {
			startIndex = dec.InputOffset()
		}

	}
	return extension, nil
}
