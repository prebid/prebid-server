/*
	type-bid-assignment tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for _, sb := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					// ruleid: type-bid-assignment-check
					Bid:     &sb,
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for _, sb := range seatBid.Bid {
			sbcopy := sb
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					// ok: type-bid-assignment-check
					Bid:     &sbcopy,
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for _, sb := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				return nil, err
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				// ruleid: type-bid-assignment-check
				Bid:     &sb,
				BidType: bidType,
			})

		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for _, sb := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				return nil, err
			}
			// ruleid: type-bid-assignment-check
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: &sb, BidType: bidType})
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var errors []error
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			var t adapters.TypedBid
			// ruleid: type-bid-assignment-check
			t.Bid = &bid
			bidResponse.Bids = append(bidResponse.Bids, &t)
		}
	}
	return bidResponse, errors
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var errors []error
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			var t adapters.TypedBid
			t = adapters.TypedBid{
				// ruleid: type-bid-assignment-check
				Bid: &bid,
			}

			bidResponse.Bids = append(bidResponse.Bids, &t)
		}
	}
	return bidResponse, errors
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for idx, _ := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					// ok: type-bid-assignment-check
					Bid:     &seatBid.Bid[idx],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	for _, seatBid := range bidResp.SeatBid {
		for idx := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				return nil, err
			}
			// ok: type-bid-assignment-check
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{Bid: &seatBid.Bid[idx], BidType: bidType})
		}
	}
}
