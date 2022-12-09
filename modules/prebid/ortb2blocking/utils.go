package ortb2blocking

import "strings"

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
