package macros

type Replacer interface {
	// Replace the macros and returns replaced string
	// if any error the error will be returned
	Replace(url string, macroProvider *macroProvider) (string, error)
}
