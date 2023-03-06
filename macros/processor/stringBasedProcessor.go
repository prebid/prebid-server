package processor

import (
	"bytes"
	"strings"
	"sync"

	"github.com/prebid/prebid-server/config"
)

type stringBasedProcessor struct {
	cfg       config.MacroProcessorConfig
	templates map[string]urlMetaTemplate
	sync.RWMutex
}

func newStringBasedProcessor(cfg config.MacroProcessorConfig) *stringBasedProcessor {
	return &stringBasedProcessor{
		cfg:       cfg,
		templates: make(map[string]urlMetaTemplate),
	}
}

type urlMetaTemplate struct {
	indices     []int
	macroLength []int
}

func constructTemplate(url string, delimiter string) urlMetaTemplate {
	currentIndex := 0
	tmplt := urlMetaTemplate{
		indices:     []int{},
		macroLength: []int{},
	}
	delimiterLen := len(delimiter)
	for {
		currentIndex = currentIndex + strings.Index(url[currentIndex:], delimiter)
		if currentIndex == -1 {
			break
		}
		middleIndex := currentIndex + delimiterLen
		endingIndex := strings.Index(url[middleIndex:], delimiter) // ending Delimiter
		if endingIndex == -1 {
			break
		}
		endingIndex = endingIndex + middleIndex // offset adjustment (Delimiter inclusive)
		macroLength := endingIndex              // just for readiability
		tmplt.indices = append(tmplt.indices, currentIndex)
		tmplt.macroLength = append(tmplt.macroLength, macroLength)
		currentIndex = endingIndex + 1
		if currentIndex >= len(url) {
			break
		}
	}
	return tmplt
}

func (processor *stringBasedProcessor) Replace(url string, macroProvider Provider) (string, error) {
	tmplt := processor.getTemplate(url)

	var result bytes.Buffer
	// iterate over macros startindex list to get position where value should be put
	// http://tracker.com?macro_1=##PBS_EVENTTYPE##&macro_2=##PBS_GDPRCONSENT##&custom=##PBS_MACRO_profileid##&custom=##shri##
	currentIndex := 0
	delimLen := len(processor.cfg.Delimiter)
	for i, index := range tmplt.indices {
		macro := url[index+delimLen : tmplt.macroLength[i]]
		// copy prev part
		result.WriteString(url[currentIndex:index])
		value := macroProvider.GetMacro(macro)
		if value != "" {
			result.WriteString(value)
		}
		currentIndex = index + len(macro) + 2*delimLen
	}
	result.WriteString(url[currentIndex:])
	return result.String(), nil
}

func (processor *stringBasedProcessor) getTemplate(url string) urlMetaTemplate {
	var (
		template urlMetaTemplate
		ok       bool
	)
	processor.RLock()
	template, ok = processor.templates[url]
	processor.RUnlock()

	if !ok {
		processor.Lock()
		template = constructTemplate(url, processor.cfg.Delimiter)
		processor.templates[url] = template
		processor.Unlock()
	}
	return template
}
