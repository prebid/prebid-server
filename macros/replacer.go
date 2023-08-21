package macros

import "strings"

type Replacer interface {
	// Replace the macros and returns replaced string
	// if any error the error will be returned
	Replace(url string, macroProvider *MacroProvider) (string, error)
	ReplaceBytes(buf *strings.Builder, url string, macroProvider *MacroProvider)
}
