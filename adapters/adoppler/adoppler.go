package adoppler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var bidHeaders http.Header = map[string][]string{
	"Accept":            {"application/json"},
	"Content-Type":      {"application/json;charset=utf-8"},
	"X-OpenRTB-Version": {"2.5"},
}

type adsVideoExt struct {
	Duration int `json:"duration"`
}

type adsImpExt struct {
	Video *adsVideoExt `json:"video"`
}

type AdopplerAdapter struct {
	endpoint string
}

func NewAdopplerBidder(endpoint string) *AdopplerAdapter {
	return &AdopplerAdapter{endpoint}
}

func (ads *AdopplerAdapter) MakeRequests(
	req *openrtb.BidRequest,
	info *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	if len(req.Imp) == 0 {
		return nil, nil
	}

	var datas []*adapters.RequestData
	var errs []error
	for _, imp := range req.Imp {
		ext, err := unmarshalExt(imp.Ext)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{err.Error()})
			continue
		}

		var r openrtb.BidRequest = *req
		r.ID = req.ID + "-" + ext.AdUnit
		r.Imp = []openrtb.Imp{imp}

		body, err := json.Marshal(r)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := fmt.Sprintf("%s/processHeaderBid/%s",
			ads.endpoint, url.PathEscape(ext.AdUnit))
		data := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    body,
			Headers: bidHeaders,
		}
		datas = append(datas, data)
	}

	return datas, errs
}

func (ads *AdopplerAdapter) MakeBids(
	intReq *openrtb.BidRequest,
	extReq *adapters.RequestData,
	resp *adapters.ResponseData,
) (
	*adapters.BidderResponse,
	[]error,
) {
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{"bad request"}}
	}
	if resp.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			fmt.Sprintf("unexpected status: %d", resp.StatusCode),
		}
		return nil, []error{err}
	}

	var bidResp openrtb.BidResponse
	err := json.Unmarshal(resp.Body, &bidResp)
	if err != nil {
		err := &errortypes.BadServerResponse{
			fmt.Sprintf("invalid body: %s", err.Error()),
		}
		return nil, []error{err}
	}

	impTypes := make(map[string]openrtb_ext.BidType)
	for _, imp := range intReq.Imp {
		if _, ok := impTypes[imp.ID]; ok {
			return nil, []error{&errortypes.BadInput{
				fmt.Sprintf("duplicate $.imp.id %s", imp.ID),
			}}
		}
		if imp.Banner != nil {
			impTypes[imp.ID] = openrtb_ext.BidTypeBanner
		} else if imp.Video != nil {
			impTypes[imp.ID] = openrtb_ext.BidTypeVideo
		} else if imp.Audio != nil {
			impTypes[imp.ID] = openrtb_ext.BidTypeAudio
		} else if imp.Native != nil {
			impTypes[imp.ID] = openrtb_ext.BidTypeNative
		} else {
			return nil, []error{&errortypes.BadInput{
				"one of $.imp.banner, $.imp.video, $.imp.audio and $.imp.native field required",
			}}
		}
	}

	var bids []*adapters.TypedBid
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			tp, ok := impTypes[bid.ImpID]
			if !ok {
				err := &errortypes.BadServerResponse{
					fmt.Sprintf("unknown impid: %s", bid.ImpID),
				}
				return nil, []error{err}
			}

			var bidVideo *openrtb_ext.ExtBidPrebidVideo
			if tp == openrtb_ext.BidTypeVideo {
				adsExt, err := unmarshalAdsExt(bid.Ext)
				if err != nil {
					return nil, []error{&errortypes.BadServerResponse{err.Error()}}
				}
				if adsExt == nil || adsExt.Video == nil {
					return nil, []error{&errortypes.BadServerResponse{
						"$.seatbid.bid.ext.ads.video required",
					}}
				}
				bidVideo = &openrtb_ext.ExtBidPrebidVideo{
					Duration:        adsExt.Video.Duration,
					PrimaryCategory: head(bid.Cat),
				}
			}
			bids = append(bids, &adapters.TypedBid{
				Bid:      &bid,
				BidType:  tp,
				BidVideo: bidVideo,
			})
		}
	}

	adsResp := adapters.NewBidderResponseWithBidsCapacity(len(bids))
	adsResp.Bids = bids

	return adsResp, nil
}

func unmarshalExt(ext json.RawMessage) (*openrtb_ext.ExtImpAdoppler, error) {
	var bext adapters.ExtImpBidder
	err := json.Unmarshal(ext, &bext)
	if err != nil {
		return nil, err
	}

	var adsExt openrtb_ext.ExtImpAdoppler
	err = json.Unmarshal(bext.Bidder, &adsExt)
	if err != nil {
		return nil, err
	}

	if adsExt.AdUnit == "" {
		return nil, errors.New("$.imp.ext.adoppler.adunit required")
	}

	return &adsExt, nil
}

func unmarshalAdsExt(ext json.RawMessage) (*adsImpExt, error) {
	var e struct {
		Ads *adsImpExt `json:"ads"`
	}
	err := json.Unmarshal(ext, &e)

	return e.Ads, err
}

func head(s []string) string {
	if len(s) == 0 {
		return ""
	}

	return s[0]
}
