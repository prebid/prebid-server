package adapters

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// PlacementType ...
type PlacementType string

const (
	Interstitial PlacementType = "interstitial"
	Rewarded     PlacementType = "rewarded"
)

func ExtractReqExtBidderParamsMap(bidRequest *openrtb2.BidRequest) (map[string]json.RawMessage, error) {
	if bidRequest == nil {
		return nil, errors.New("error bidRequest should not be nil")
	}

	reqExt := &openrtb_ext.ExtRequest{}
	if len(bidRequest.Ext) > 0 {
		err := json.Unmarshal(bidRequest.Ext, &reqExt)
		if err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]json.RawMessage
	err := json.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams)
	if err != nil {
		return nil, err
	}

	return bidderParams, nil
}

func ExtractReqExtBidderParamsEmbeddedMap(bidRequest *openrtb2.BidRequest) (map[string]map[string]json.RawMessage, error) {
	if bidRequest == nil {
		return nil, errors.New("error bidRequest should not be nil")
	}

	reqExt := &openrtb_ext.ExtRequest{}
	if len(bidRequest.Ext) > 0 {
		if err := json.Unmarshal(bidRequest.Ext, &reqExt); err != nil {
			return nil, fmt.Errorf("error decoding Request.ext : %s", err.Error())
		}
	}

	if reqExt.Prebid.BidderParams == nil {
		return nil, nil
	}

	var bidderParams map[string]map[string]json.RawMessage
	if err := json.Unmarshal(reqExt.Prebid.BidderParams, &bidderParams); err != nil {
		return nil, err
	}

	return bidderParams, nil
}

// FilterPrebidSKADNExt -- Added by Tapjoy to handle SKADN DSP extensions
// returns filtered openrtb_ext.SKADN extension object filtered by map
func FilterPrebidSKADNExt(prebidExt *openrtb_ext.ExtImpPrebid, filterMap map[string]bool) openrtb_ext.SKADN {
	if prebidExt == nil {
		return openrtb_ext.SKADN{}
	}

	return openrtb_ext.SKADN{
		Version:    prebidExt.SKADN.Version,
		Versions:   prebidExt.SKADN.Versions,
		SourceApp:  prebidExt.SKADN.SourceApp,
		SKADNetIDs: filterArrayWithMap(prebidExt.SKADN.SKADNetIDs, filterMap),
	}
}

// filterArrayWithMap -- Added by Tapjoy to handle SKADN DSP filtering
// returns a subset elements of arr whose keys were in filterMap
func filterArrayWithMap(arr []string, filterMap map[string]bool) (ret []string) {
	for _, id := range arr {
		if _, ok := filterMap[strings.ToLower(id)]; ok {
			ret = append(ret, id)
		}
	}
	return ret
}
