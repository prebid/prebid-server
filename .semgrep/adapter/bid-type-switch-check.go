/*
	bid-type-switch-check tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

// ruleid: bid-type-switch-check
switch bidExt.AdCodeType {
case "banner":
	return openrtb_ext.BidTypeBanner, nil
case "native":
	return openrtb_ext.BidTypeNative, nil
case "video":
	return openrtb_ext.BidTypeVideo, nil
}

// ruleid: bid-type-switch-check
switch impExt.Adot.MediaType {
case string(openrtb_ext.BidTypeBanner):
	return openrtb_ext.BidTypeBanner, nil
case string(openrtb_ext.BidTypeVideo):
	return openrtb_ext.BidTypeVideo, nil
case string(openrtb_ext.BidTypeNative):
	return openrtb_ext.BidTypeNative, nil
}

// ok: bid-type-switch-check
switch bid.MType {
case "banner":
	return openrtb_ext.BidTypeBanner, nil
case "native":
	return openrtb_ext.BidTypeNative, nil
case "video":
	return openrtb_ext.BidTypeVideo, nil
}
