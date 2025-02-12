package smaato

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractAdmBanner(t *testing.T) {
	tests := []struct {
		testName         string
		adMarkup         string
		curls            []string
		expectedAdMarkup string
	}{
		{
			testName:         "extract_banner_without_curls",
			adMarkup:         `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			expectedAdMarkup: `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			curls:            []string{},
		},
		{
			testName:         "extract_banner_with_nil_curls",
			adMarkup:         `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			expectedAdMarkup: `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			curls:            nil,
		},
		{
			testName:         "extract_banner_with_curls",
			adMarkup:         `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			expectedAdMarkup: `<div style="cursor:pointer" onclick="fetch(decodeURIComponent('curls.net'.replace(/\+/g, ' ')), {cache: 'no-cache'});"><a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a></div>`,
			curls:            []string{"curls.net"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			adMarkup := extractAdmBanner(tt.adMarkup, tt.curls)

			assert.Equal(t, tt.expectedAdMarkup, adMarkup)
		})
	}
}
