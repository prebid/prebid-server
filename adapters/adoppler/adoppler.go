package adoppler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const DefaultClient = "app"

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
	endpoint *template.Template
}

// Builder builds a new instance of the Adoppler adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &AdopplerAdapter{
		endpoint: template,
	}
	return bidder, nil
}

func (ads *AdopplerAdapter) MakeRequests(
	req *openrtb2.BidRequest,
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

		var r openrtb2.BidRequest = *req
		r.ID = req.ID + "-" + ext.AdUnit
		r.Imp = []openrtb2.Imp{imp}

		body, err := json.Marshal(r)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri, err := ads.bidUri(ext)
		if err != nil {
			e := fmt.Sprintf("Unable to build bid URI: %s",
				err.Error())
			errs = append(errs, &errortypes.BadInput{e})
			continue
		}
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
	intReq *openrtb2.BidRequest,
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

	var bidResp openrtb2.BidResponse
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

func (ads *AdopplerAdapter) bidUri(ext *openrtb_ext.ExtImpAdoppler) (string, error) {
	params := macros.EndpointTemplateParams{}
	params.AdUnit = url.PathEscape(ext.AdUnit)
	if ext.Client == "" {
		params.AccountID = DefaultClient
	} else {
		params.AccountID = url.PathEscape(ext.Client)
	}

	return macros.ResolveMacros(*ads.endpoint, params)
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
