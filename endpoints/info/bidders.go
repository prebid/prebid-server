package info

import (
	"encoding/json"
	"net/http"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// NewBiddersEndpoint implements /info/bidders
func NewBiddersEndpoint(aliases map[string]string) httprouter.Handle {
	bidderNames := make([]string, 0, len(openrtb_ext.BidderMap)+len(aliases))
	for bidderName := range openrtb_ext.BidderMap {
		bidderNames = append(bidderNames, bidderName)
	}

	for aliasName := range aliases {
		bidderNames = append(bidderNames, aliasName)
	}

	biddersJson, err := json.Marshal(bidderNames)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint response: %v", err)
	}

	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(biddersJson); err != nil {
			glog.Errorf("error writing response to /info/bidders: %v", err)
		}
	})
}

// NewBiddersEndpoint implements /info/bidders/*
func NewBidderDetailsEndpoint(infos adapters.BidderInfos, aliases map[string]string) httprouter.Handle {
	// Build all the responses up front, since there are a finite number and it won't use much memory.
	responses := make(map[string]json.RawMessage, len(infos))
	for bidderName, bidderInfo := range infos {
		jsonData, err := json.Marshal(bidderInfo)
		if err != nil {
			glog.Fatalf("Failed to JSON-marshal bidder-info/%s.yaml data.", bidderName)
		}
		responses[bidderName] = jsonData
	}

	// Add in any default aliases
	for aliasName, bidderName := range aliases {
		responses[aliasName] = createAliasInfo(responses, aliasName, bidderName)
	}

	// Return an endpoint which writes the responses from memory.
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		forBidder := ps.ByName("bidderName")
		if response, ok := responses[forBidder]; ok {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(response); err != nil {
				glog.Errorf("error writing response to /info/bidders/%s: %v", forBidder, err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func createAliasInfo(responses map[string]json.RawMessage, alias string, core string) json.RawMessage {
	coreJSON, ok := responses[core]
	if !ok {
		glog.Fatalf("Unknown core bidder %s for default alias %s", core, alias)
	}
	jsonData := make(json.RawMessage, len(coreJSON))
	copy(jsonData, coreJSON)

	jsonInfo, err := jsonparser.Set(jsonData, []byte(`"`+core+`"`), "aliasOf")
	if err != nil {
		glog.Fatalf("Failed to generate bidder info for %s, an alias of %s", alias, core)
	}
	return jsonInfo
}

type infoFile struct {
	Maintainer   *maintainerInfo   `yaml:"maintainer" json:"maintainer"`
	Capabilities *capabilitiesInfo `yaml:"capabilities" json:"capabilities"`
}

type maintainerInfo struct {
	Email string `yaml:"email" json:"email"`
}

type capabilitiesInfo struct {
	App  *platformInfo `yaml:"app" json:"app"`
	Site *platformInfo `yaml:"site" json:"site"`
}

type platformInfo struct {
	MediaTypes []string `yaml:"mediaTypes" json:"mediaTypes"`
}
