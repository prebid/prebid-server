package macros

import "strings"

type Replacer interface {
	// Replace the macros and returns replaced string
	// if any error the error will be returned
	Replace(result *strings.Builder, url string, macroProvider *MacroProvider)
}
