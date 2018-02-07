package exchange

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
)

// cleanOpenRTBRequests splits the input request into requests which are sanitized for each bidder. Intended behavior is:
//
//   1. BidRequest.Imp[].Ext will only contain the "prebid" field and a "bidder" field which has the params for the intended Bidder.
//   2. Every BidRequest.Imp[] requested Bids from the Bidder who keys it.
//   3. BidRequest.User.BuyerUID will be set to that Bidder's ID.
func cleanOpenRTBRequests(orig *openrtb.BidRequest, usersyncs IdFetcher, met *pbsmetrics.Metrics) (requestsByBidder map[openrtb_ext.BidderName]*openrtb.BidRequest, aliases map[string]string, errs []error) {
	impsByBidder, errs := splitImps(orig.Imp)
	if len(errs) > 0 {
		return
	}

	aliases, errs = parseAliases(orig)
	if len(errs) > 0 {
		return
	}

	requestsByBidder = splitBidRequest(orig, impsByBidder, aliases, usersyncs, met)
	return
}

func splitBidRequest(req *openrtb.BidRequest, impsByBidder map[string][]openrtb.Imp, aliases map[string]string, usersyncs IdFetcher, met *pbsmetrics.Metrics) map[openrtb_ext.BidderName]*openrtb.BidRequest {
	requestsByBidder := make(map[openrtb_ext.BidderName]*openrtb.BidRequest, len(impsByBidder))
	for bidder, imps := range impsByBidder {
		reqCopy := *req
		coreBidder := resolveBidder(bidder, aliases)
		met.AdapterMetrics[coreBidder].RequestMeter.Mark(1)
		if hadSync := prepareUser(&reqCopy, coreBidder, usersyncs); !hadSync && req.App == nil {
			met.AdapterMetrics[coreBidder].NoCookieMeter.Mark(1)
		}
		reqCopy.Imp = imps
		requestsByBidder[openrtb_ext.BidderName(bidder)] = &reqCopy
	}
	return requestsByBidder
}

// splitImps takes a list of Imps and returns a map of imps which have been sanitized for each bidder.
//
// For example, suppose imps has two elements. One goes to rubicon, while the other goes to appnexus and index.
// The returned map will have three keys: rubicon, appnexus, and index--each with one Imp.
// The "imp.ext" value of the appnexus Imp will only contain the "prebid" values, and "appnexus" value at the "bidder" key.
// The "imp.ext" value of the rubicon Imp will only contain the "prebid" values, and "rubicon" value at the "bidder" key.
//
// The goal here is so that Bidders only get Imps and Imp.Ext values which are intended for them.
func splitImps(imps []openrtb.Imp) (map[string][]openrtb.Imp, []error) {
	impExts, err := parseImpExts(imps)
	if err != nil {
		return nil, []error{err}
	}

	splitImps := make(map[string][]openrtb.Imp, len(imps))
	var errList []error
	for i := 0; i < len(imps); i++ {
		thisImp := imps[i]
		theseBidders := impExts[i]
		for intendedBidder := range theseBidders {
			if intendedBidder == "prebid" {
				continue
			}

			otherImps, _ := splitImps[intendedBidder]
			if impForBidder, err := sanitizedImpCopy(&thisImp, theseBidders, intendedBidder); err != nil {
				errList = append(errList, err)
			} else {
				splitImps[intendedBidder] = append(otherImps, *impForBidder)
			}
		}
	}

	return splitImps, nil
}

// sanitizedImpCopy returns a copy of imp with its ext filtered so that only "prebid" and intendedBidder exist.
// It will not mutate the input imp.
// This function expects the "ext" argument to have been unmarshalled from "imp", so we don't have to repeat that work.
func sanitizedImpCopy(imp *openrtb.Imp, ext map[string]openrtb.RawJSON, intendedBidder string) (*openrtb.Imp, error) {
	impCopy := *imp
	newExt := make(map[string]openrtb.RawJSON, 2)
	if value, ok := ext["prebid"]; ok {
		newExt["prebid"] = value
	}
	newExt["bidder"] = ext[intendedBidder]
	extBytes, err := json.Marshal(newExt)
	if err != nil {
		return nil, err
	}
	impCopy.Ext = extBytes
	return &impCopy, nil
}

// prepareUser changes req.User so that it's ready for the given bidder.
// This *will* mutate the request, but will *not* mutate any objects nested inside it.
//
// This function expects bidder to be a "known" bidder name. It will not work on aliases.
// It returns true if an ID sync existed, and false otherwise.
func prepareUser(req *openrtb.BidRequest, bidder openrtb_ext.BidderName, usersyncs IdFetcher) bool {
	if id, ok := usersyncs.GetId(bidder); ok {
		if req.User == nil {
			req.User = &openrtb.User{
				BuyerUID: id,
			}
		} else if req.User.BuyerUID == "" {
			clone := *req.User
			clone.BuyerUID = id
			req.User = &clone
		}
		return true
	}
	return false
}

// resolveBidder returns the known BidderName associated with bidder, if bidder is an alias. If it's not an alias, the bidder is returned.
func resolveBidder(bidder string, aliases map[string]string) openrtb_ext.BidderName {
	if coreBidder, ok := aliases[bidder]; ok {
		return openrtb_ext.BidderName(coreBidder)
	}
	return openrtb_ext.BidderName(bidder)
}

// parseImpExts does a partial-unmarshal of the imp[].Ext field.
// The keys in the returned map are expected to be "prebid", core BidderNames, or Aliases for this request.
func parseImpExts(imps []openrtb.Imp) ([]map[string]openrtb.RawJSON, error) {
	exts := make([]map[string]openrtb.RawJSON, len(imps))
	// Loop over every impression in the request
	for i := 0; i < len(imps); i++ {
		// Unpack each set of extensions found in the Imp array
		err := json.Unmarshal(imps[i].Ext, &exts[i])
		if err != nil {
			return nil, fmt.Errorf("Error unpacking extensions for Imp[%d]: %s", i, err.Error())
		}
	}
	return exts, nil
}

// parseAliases parses the aliases from the BidRequest
func parseAliases(orig *openrtb.BidRequest) (map[string]string, []error) {
	var aliases map[string]string
	if value, dataType, _, err := jsonparser.Get(orig.Ext, "prebid", "aliases"); dataType == jsonparser.Object && err == nil {
		if err := json.Unmarshal(value, &aliases); err != nil {
			return nil, []error{err}
		}
	} else if dataType != jsonparser.NotExist && err != jsonparser.KeyPathNotFoundError {
		return nil, []error{err}
	}
	return aliases, nil
}

// Quick little randomizer for a list of strings. Stuffing it in utils to keep other files clean
func randomizeList(list []openrtb_ext.BidderName) {
	l := len(list)
	perm := rand.Perm(l)
	var j int
	for i := 0; i < l; i++ {
		j = perm[i]
		list[i], list[j] = list[j], list[i]
	}
}
