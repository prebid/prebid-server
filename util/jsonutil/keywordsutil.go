package jsonutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Keywords string

// KeyVals defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus.keywords[i]
type KeyVals struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

func (ks *Keywords) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	switch b[0] {
	case '{':
		var results map[string][]string
		if err := json.Unmarshal(b, &results); err != nil {
			return err
		}

		var keywords strings.Builder
		for key, values := range results {
			for _, val := range values {
				keywords.WriteString(fmt.Sprintf("%s=%s,", key, val))
			}
		}
		*ks = Keywords(keywords.String()[:keywords.Len()-1])
	case '[':
		var results []KeyVals
		if err := json.Unmarshal(b, &results); err != nil {
			return err
		}
		var kvs strings.Builder
		for _, kv := range results {
			if len(kv.Values) == 0 {
				kvs.WriteString(fmt.Sprintf("%s,", kv.Key))
			} else {
				for _, val := range kv.Values {
					kvs.WriteString(fmt.Sprintf("%s=%s,", kv.Key, val))
				}
			}
		}

		*ks = Keywords(kvs.String()[:kvs.Len()-1])
	case '"':
		var keywords string
		if err := json.Unmarshal(b, &keywords); err != nil {
			return err
		}
		*ks = Keywords(keywords)
	}
	return nil
}

func (ks *Keywords) String() string {
	return *(*string)(ks)
}
