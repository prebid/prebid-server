package smaato

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractAdmImage(t *testing.T) {
	tests := []struct {
		testName         string
		adMarkup         string
		curls            []string
		expectedAdMarkup string
	}{
		{
			testName:         "extract banner without curls",
			adMarkup:         `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			expectedAdMarkup: `<div style="cursor:pointer" ><a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a></div>`,
			curls:            []string{},
		},
		{
			testName:         "extract banner with curls",
			adMarkup:         `<a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a>`,
			expectedAdMarkup: `<div style="cursor:pointer" fetch(decodeURIComponent('curls.net'.replace(/\+/g, ' ')), {cache: 'no-cache'});><a rel="nofollow" href="https://prebid.net/click"><img src="https://prebid.net/images/image.png" alt="" width="480" height="320" /></a></div>`,
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
