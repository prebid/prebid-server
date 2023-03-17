package jsonutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Keywords string

// extImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus.keywords[i]
type extImpAppnexusKeyVal struct {
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
		err := json.Unmarshal(b, &results)
		if err != nil {
			return err
		}

		var keywords string
		for key, values := range results {
			for _, val := range values {
				keywords = keywords + fmt.Sprintf("%s=%s,", key, val)
			}
		}
		*ks = Keywords(strings.TrimSuffix(keywords, ","))
	case '[':
		var results []extImpAppnexusKeyVal
		err := json.Unmarshal(b, &results)
		if err != nil {
			return err
		}
		var kvs string
		for _, kv := range results {
			if len(kv.Values) == 0 {
				kvs = kvs + fmt.Sprintf("%s,", kv.Key)
			} else {
				for _, val := range kv.Values {
					kvs = kvs + fmt.Sprintf("%s=%s,", kv.Key, val)
				}
			}
		}

		*ks = Keywords(strings.TrimSuffix(kvs, ","))
	case '"':
		*ks = Keywords(string(b[1 : len(b)-1]))
	}
	return nil
}
