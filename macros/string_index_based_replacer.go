package macros

import (
	"strings"
	"sync"
)

const (
	delimiter = "##"
)

type stringIndexBasedReplacer struct {
	templates map[string]urlMetaTemplate
	sync.RWMutex
}

type urlMetaTemplate struct {
	startingIndices []int
	endingIndices   []int
}

// NewStringIndexBasedReplacer will return instance of string index based macro replacer
func NewStringIndexBasedReplacer() Replacer {
	return &stringIndexBasedReplacer{
		templates: make(map[string]urlMetaTemplate),
	}
}

// constructTemplate func finds index bounds of all macros in an input string where macro format is ##data##.
// constructTemplate func returns two arrays with start indexes and end indexes for all macros found in the input string.
// Start index of the macro points to the index of the delimiter(##) start.
// End index of the macro points to the end index of the delimiter.
// For the valid input string number of start and end indexes should be equal, and they should not intersect.
// This approach shows better performance results compare to standard GoLang string replacer.
func constructTemplate(url string) urlMetaTemplate {
	currentIndex := 0
	tmplt := urlMetaTemplate{
		startingIndices: []int{},
		endingIndices:   []int{},
	}
	delimiterLen := len(delimiter)
	for {
		currentIndex = currentIndex + strings.Index(url[currentIndex:], delimiter)
		if currentIndex == -1 {
			break
		}
		startIndex := currentIndex + delimiterLen
		endingIndex := strings.Index(url[startIndex:], delimiter)
		if endingIndex == -1 {
			break
		}
		endingIndex = endingIndex + startIndex - 1
		tmplt.startingIndices = append(tmplt.startingIndices, startIndex)
		tmplt.endingIndices = append(tmplt.endingIndices, endingIndex)
		currentIndex = endingIndex + delimiterLen + 1
		if currentIndex >= len(url)-1 {
			break
		}
	}
	return tmplt
}

// Replace function replaces macros in a given string with the data from macroProvider and returns modified input string.
// If a given string was previously processed this function fetches its metadata from the cache.
// If input string is not found in cache then template metadata will be created.
// Iterates over start and end indexes of the template arrays and extracts macro name from the input string.
// Gets the value of the extracted macro from the macroProvider. Replaces macro with corresponding value.
func (s *stringIndexBasedReplacer) Replace(result *strings.Builder, url string, macroProvider *MacroProvider) {
	template := s.getTemplate(url)
	currentIndex := 0
	delimLen := len(delimiter)
	for i, index := range template.startingIndices {
		macro := url[index : template.endingIndices[i]+1]
		// copy prev part
		result.WriteString(url[currentIndex : index-delimLen])
		value := macroProvider.GetMacro(macro)
		if value != "" {
			result.WriteString(value)
		}
		currentIndex = index + len(macro) + delimLen
	}
	result.WriteString(url[currentIndex:])
}

func (s *stringIndexBasedReplacer) getTemplate(url string) urlMetaTemplate {
	var (
		template urlMetaTemplate
		ok       bool
	)
	s.RLock()
	template, ok = s.templates[url]
	s.RUnlock()

	if !ok {
		s.Lock()
		template = constructTemplate(url)
		s.templates[url] = template
		s.Unlock()
	}
	return template
}
