/*
	parse-bid-type-check tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			// ruleid: parse-bid-type-check
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	var bidExt bidExt
	// ruleid: parse-bid-type-check
	bidType, err := openrtb_ext.ParseBidType(bidExt.Prebid.Type)

	return bidType, err
}
