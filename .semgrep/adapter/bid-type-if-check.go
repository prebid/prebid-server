/*
	bid-type-if-check tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			// ruleid: bid-type-if-check
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
				// ruleid: bid-type-if-check
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
				// ruleid: bid-type-if-check
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
				// ruleid: bid-type-if-check
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			// ruleid: bid-type-if-check
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find native/banner/video impression \"%s\" ", impID),
	}
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			// ruleid: bid-type-if-check
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			}
		}
	}
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find native/banner/video impression \"%s\" ", impID),
	}
}
