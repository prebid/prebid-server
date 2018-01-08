package exchange

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"math/rand"
)

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

// This will copy the openrtb BidRequest into an array of requests, where the BidRequest.Imp[].Ext field will only
// consist of the "prebid" field and the field for the appropriate bidder parameters. We will drop all extended fields
// beyond this context, so this will not be compatible with any other uses of the extension area. That is, this routine
// will work, but the adapters will not see any other extension fields.

// NOTE: the return map will only contain entries for bidders that both have the extension field in at least one Imp,
// and are listed in the adapters string. wseats and bseats can be implimented by passing the bseats list as adapters,
// or after return removing any adapters listed in wseats. Or removing all adapters in wseats from the adapters list
// before submitting.

// Take an openrtb request, and a list of bidders, and return an openrtb request sanitized for each bidder
func cleanOpenRTBRequests(orig *openrtb.BidRequest, adapters []openrtb_ext.BidderName, usersyncs IdFetcher) (map[openrtb_ext.BidderName]*openrtb.BidRequest, []error) {
	// This is the clean array of openrtb requests we will be returning
	cleanReqs := make(map[openrtb_ext.BidderName]*openrtb.BidRequest, len(adapters))
	errList := make([]error, 0, 1)

	// Decode the Imp extensions once to save time. We store the results here
	imp_exts := make([]map[string]openrtb.RawJSON, len(orig.Imp))
	// Loop over every impression in the request
	for i := 0; i < len(orig.Imp); i++ {
		// Unpack each set of extensions found in the Imp array
		err := json.Unmarshal(orig.Imp[i].Ext, &imp_exts[i])
		if err != nil {
			return nil, []error{fmt.Errorf("Error unpacking extensions for Imp[%d]: %s", i, err.Error())}
		}
	}

	// Loop over every adapter we want to create a clean openrtb request for.
	for i := 0; i < len(adapters); i++ {
		// Go deeper into Imp array
		newImps := make([]openrtb.Imp, 0, len(orig.Imp))
		bn := adapters[i].String()

		// Overwrite each extension field with a cleanly built subset
		// We are looping over every impression in the Imp array
		for j := 0; j < len(orig.Imp); j++ {
			// Don't do anything if the current bidder's field is not present.
			if val, ok := imp_exts[j][bn]; ok {
				// Start with a new, empty unpacked extention
				newExts := make(map[string]openrtb.RawJSON, len(orig.Imp))
				// Need to do some consistency checking to verify these fields exist. Especially the adapters one.
				if pb, ok := imp_exts[j]["prebid"]; ok {
					newExts["prebid"] = pb
				}
				newExts["bidder"] = val
				// Create a "clean" byte array for this Imp's extension
				// Note, if the "prebid" or "<adapter>" field is missing from the source, it will be missing here as well
				// The adapters should test that their field is present rather than assuming it will be there if they are
				// called
				b, err := json.Marshal(newExts)
				if err != nil {
					errList = append(errList, fmt.Errorf("Error creating sanitized bidder extents for Imp[%d], bidder %s: %s", j, bn, err.Error()))
				}
				// Overwrite the extention field with the new cleaned version
				newImps = append(newImps, orig.Imp[j])
				newImps[len(newImps)-1].Ext = b
			}
		}

		// Only add a BidRequest if there exist Imp(s) for this adapter
		if len(newImps) > 0 {
			// Create a new BidRequest
			newReq := new(openrtb.BidRequest)
			// Make a shallow copy of the original request
			*newReq = *orig
			prepareUser(newReq, adapters[i], usersyncs)
			newReq.Imp = newImps
			cleanReqs[adapters[i]] = newReq
		}
	}
	return cleanReqs, errList
}

// prepareUser changes req.User so that it's ready for the given bidder.
// This *will* mutate the request, but will *not* mutate any objects nested inside it.
func prepareUser(req *openrtb.BidRequest, bidder openrtb_ext.BidderName, usersyncs IdFetcher) {
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
	}
}

func makeLoggableAdapterRequests(name openrtb_ext.BidderName, reqData []*adapters.RequestData) []analytics.LoggableAdapterRequests {
	ar := make([]analytics.LoggableAdapterRequests, len(reqData))
	for i, req := range reqData {
		ar[i] = analytics.LoggableAdapterRequests{
			Name:     string(name),
			Requests: string(req.Body),
			Uri:      req.Uri,
			Method:   req.Method,
			Header:   req.Headers,
		}
	}
	return ar
}
