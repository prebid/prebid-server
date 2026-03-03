package ortb2blocking

import (
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func mergeStrings(messages []string, newMessages ...string) []string {
	for _, msg := range newMessages {
		if msg == "" {
			continue
		}
		messages = append(messages, msg)
	}
	return messages
}

func hasMatches(list []string, s string) bool {
	for _, val := range list {
		if strings.EqualFold(val, s) {
			return true
		}
	}
	return false
}

type numeric interface {
	openrtb2.BannerAdType | adcom1.CreativeAttribute
}

func toInt[T numeric](values []T) []int {
	ints := make([]int, len(values))
	for i := range values {
		ints[i] = int(values[i])
	}
	return ints
}
