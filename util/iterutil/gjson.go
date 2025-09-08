package iterutil

import (
	"iter"

	"github.com/tidwall/gjson"
)

type (
	GjsonElem struct {
		path   string
		result gjson.Result
	}
)

// WalkGjsonLeaves returns an iterator of (path, result) from a gjson result; only leaves are yielded.
func WalkGjsonLeaves(result gjson.Result) iter.Seq2[string, gjson.Result] {
	return func(yield func(string, gjson.Result) bool) {
		stack := []GjsonElem{{path: "", result: result}}
		for stackLen := len(stack); stackLen > 0; stackLen = len(stack) {
			elem := stack[stackLen-1]
			stack = stack[:stackLen-1]
			if elem.result.Type == gjson.JSON {
				if elem.path != "" {
					elem.result.ForEach(func(key, value gjson.Result) bool {
						stack = append(stack, GjsonElem{path: elem.path + "." + key.String(), result: value})
						return true
					})
				} else {
					elem.result.ForEach(func(key, value gjson.Result) bool {
						stack = append(stack, GjsonElem{path: key.String(), result: value})
						return true
					})
				}
			} else if !yield(elem.path, elem.result) {
				return
			}
		}
	}
}
