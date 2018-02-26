package info

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/openrtb_ext"
	yaml "gopkg.in/yaml.v2"
)

// NewBiddersEndpoint implements /info/bidders
func NewBiddersEndpoint() httprouter.Handle {
	bidderNames := make([]string, 0, len(openrtb_ext.BidderMap))
	for bidderName := range openrtb_ext.BidderMap {
		bidderNames = append(bidderNames, bidderName)
	}

	jsonData, err := json.Marshal(bidderNames)
	if err != nil {
		glog.Fatalf("error creating /info/bidders endpoint response: %v", err)
	}

	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(jsonData); err != nil {
			glog.Errorf("error writing response to /info/bidders: %v", err)
		}
	})
}

// NewBiddersEndpoint implements /info/bidders/*
func NewBidderDetailsEndpoint(infoDir string) httprouter.Handle {
	// Build all the responses up front, since there are a finite number and it won't use much memory.
	files, err := ioutil.ReadDir(infoDir)
	if err != nil {
		glog.Fatalf("error reading directory %s: %v", infoDir, err)
	}

	responses := make(map[string]json.RawMessage, len(files))
	for _, file := range files {
		fileData, err := ioutil.ReadFile(infoDir + "/" + file.Name())
		if err != nil {
			glog.Fatalf("error reading from file %s: %v", infoDir+"/"+file.Name(), err)
		}

		var parsedInfo infoFile
		if err := yaml.Unmarshal(fileData, &parsedInfo); err != nil {
			glog.Fatalf("error parsing yaml in file %s: %v", infoDir+"/"+file.Name(), err)
		}

		jsonBytes, err := json.Marshal(parsedInfo)
		if err != nil {
			glog.Fatalf("error writing JSON of file %s: %v", infoDir+"/"+file.Name(), err)
		}
		responses[strings.TrimSuffix(file.Name(), ".yaml")] = json.RawMessage(jsonBytes)
	}

	// Return an endpoint which writes the responses as quickly as possible.
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
