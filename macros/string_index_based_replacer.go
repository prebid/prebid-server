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

// constructTemplate create a template  which store starting and ending indexes of each macro in the order they apper in event url.
// constructTemplate scans the url from start to end and looks for pair of delimiter(##).
// The string between the delimiter(##) pair is a macro key in valid event url.
// The index position after the first delimiter(##) is stored as starting index of macro and
// the index position before the second delimiter(##) is stored as ending index of the macro in the template.
// Scanning stop when start and end index position of all the macros are stored or no more macros are present in the url.
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
		endingIndex := strings.Index(url[startIndex:], delimiter) // ending Delimiter
		if endingIndex == -1 {
			break
		}
		endingIndex = endingIndex + startIndex // offset adjustment (Delimiter inclusive)
		tmplt.startingIndices = append(tmplt.startingIndices, startIndex)
		tmplt.endingIndices = append(tmplt.endingIndices, endingIndex)
		currentIndex = endingIndex + delimiterLen
		if currentIndex >= len(url)-1 {
			break
		}
	}
	return tmplt
}

// Replace will replace the macros in the given  event url.
// Check if urlMetaTemplate exist in cache for given url if not construct the template (start and end index info of each macro) and store in cache.
// Iterate over the startingIndices in the template and for each starting index and corresponding ending index, extract macro from the url.
// Get the value of the macro from the macroProvider.
// Prepend the url string present before the macro to the macro value and append the remaining url.
func (s *stringIndexBasedReplacer) Replace(url string, macroProvider *macroProvider) (string, error) {
	tmplt := s.getTemplate(url)

	var result strings.Builder
	currentIndex := 0
	delimLen := len(delimiter)
	for i, index := range tmplt.startingIndices {
		macro := url[index:tmplt.endingIndices[i]]
		// copy prev part
		result.WriteString(url[currentIndex : index-delimLen])
		value := macroProvider.GetMacro(macro)
		if value != "" {
			result.WriteString(value)
		}
		currentIndex = index + len(macro) + delimLen
	}
	result.WriteString(url[currentIndex:])
	return result.String(), nil
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
