package info

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	statusActive   string = "ACTIVE"
	statusDisabled string = "DISABLED"
)

// NewBiddersDetailEndpoint builds a handler for the /info/bidders/<bidder> endpoint.
func NewBiddersDetailEndpoint(bidders config.BidderInfos) httprouter.Handle {
	responses, err := prepareBiddersDetailResponse(bidders)
	if err != nil {
		glog.Fatalf("error creating /info/bidders/<bidder> endpoint response: %v", err)
	}

	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		bidder := ps.ByName("bidderName")
		bidderName, found := getNormalizedBidderName(bidder)

		if !found {
			w.WriteHeader(http.StatusNotFound)
		}

		if response, ok := responses[bidderName]; ok {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(response); err != nil {
				glog.Errorf("error writing response to /info/bidders/%s: %v", bidder, err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func getNormalizedBidderName(bidderName string) (string, bool) {
	if strings.ToLower(bidderName) == "all" {
		return "all", true
	}

	bidderNameNormalized, ok := openrtb_ext.NormalizeBidderName(bidderName)
	if !ok {
		return "", false
	}

	return bidderNameNormalized.String(), true
}

func prepareBiddersDetailResponse(bidders config.BidderInfos) (map[string][]byte, error) {
	details := mapDetails(bidders)

	responses, err := marshalDetailsResponse(details)
	if err != nil {
		return nil, err
	}

	all, err := marshalAllResponse(responses)
	if err != nil {
		return nil, err
	}
	responses["all"] = all

	return responses, nil
}

func mapDetails(bidders config.BidderInfos) map[string]bidderDetail {
	details := map[string]bidderDetail{}

	for bidderName, bidderInfo := range bidders {
		details[bidderName] = mapDetailFromConfig(bidderInfo)
	}

	return details
}

func marshalDetailsResponse(details map[string]bidderDetail) (map[string][]byte, error) {
	responses := map[string][]byte{}

	for bidder, detail := range details {
		json, err := jsonutil.Marshal(detail)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal info for bidder %s: %v", bidder, err)
		}
		responses[bidder] = json
	}

	return responses, nil
}

func marshalAllResponse(responses map[string][]byte) ([]byte, error) {
	responsesJSON := make(map[string]json.RawMessage, len(responses))

	for k, v := range responses {
		responsesJSON[k] = json.RawMessage(v)
	}

	json, err := jsonutil.Marshal(responsesJSON)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal info for bidder all: %v", err)
	}
	return json, nil
}

type bidderDetail struct {
	Status       string        `json:"status"`
	UsesHTTPS    *bool         `json:"usesHttps,omitempty"`
	Maintainer   *maintainer   `json:"maintainer,omitempty"`
	Capabilities *capabilities `json:"capabilities,omitempty"`
	AliasOf      string        `json:"aliasOf,omitempty"`
}

type maintainer struct {
	Email string `json:"email"`
}

type capabilities struct {
	App  *platform `json:"app,omitempty"`
	Site *platform `json:"site,omitempty"`
	DOOH *platform `json:"dooh,omitempty"`
}

type platform struct {
	MediaTypes []string `json:"mediaTypes"`
}

func mapDetailFromConfig(c config.BidderInfo) bidderDetail {
	var bidderDetail bidderDetail

	bidderDetail.AliasOf = c.AliasOf

	if c.Maintainer != nil {
		bidderDetail.Maintainer = &maintainer{
			Email: c.Maintainer.Email,
		}
	}

	if c.IsEnabled() {
		bidderDetail.Status = statusActive

		usesHTTPS := strings.HasPrefix(strings.ToLower(c.Endpoint), "https://")
		bidderDetail.UsesHTTPS = &usesHTTPS

		if c.Capabilities != nil {
			bidderDetail.Capabilities = &capabilities{}

			if c.Capabilities.App != nil {
				bidderDetail.Capabilities.App = &platform{
					MediaTypes: mapMediaTypes(c.Capabilities.App.MediaTypes),
				}
			}

			if c.Capabilities.Site != nil {
				bidderDetail.Capabilities.Site = &platform{
					MediaTypes: mapMediaTypes(c.Capabilities.Site.MediaTypes),
				}
			}

			if c.Capabilities.DOOH != nil {
				bidderDetail.Capabilities.DOOH = &platform{
					MediaTypes: mapMediaTypes(c.Capabilities.DOOH.MediaTypes),
				}
			}
		}
	} else {
		bidderDetail.Status = statusDisabled
	}

	return bidderDetail
}

func mapMediaTypes(m []openrtb_ext.BidType) []string {
	mediaTypes := make([]string, len(m))

	for i, v := range m {
		mediaTypes[i] = string(v)
	}

	return mediaTypes
}
