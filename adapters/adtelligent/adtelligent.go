package adtelligent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const uri = "http://hb.adtelligent.com/auction"

type AdtelligentAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

/* Name - export adapter name */
func (a *AdtelligentAdapter) Name() string {
	return "Adtelligent"
}

// used for cookies and such
func (a *AdtelligentAdapter) FamilyName() string {
	return "adtelligent"
}

func (a *AdtelligentAdapter) SkipNoCookies() bool {
	return false
}

type adtelligentImpExt struct {
	Adtelligent openrtb_ext.ExtImpAdtelligent `json:"adtelligent"`
}

func (a *AdtelligentAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {

	if len(request.Imp) == 0 {
		return nil, []error{
			fmt.Errorf("there are no impressions"),
		}
	}

	var err error

	if nil != err {
		return nil, []error{
			fmt.Errorf("error while encoding adtelligentImpExt, err: %s", err),
		}
	}

	errors := make([]error, 0, len(request.Imp))
	imp2source := make(map[int][]int)
	totalImps := len(request.Imp)

	for i := 0; i < totalImps; i++ {

		sourceId, err := validateImpression(&request.Imp[i])

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if _, ok := imp2source[sourceId]; !ok {
			imp2source[sourceId] = make([]int, 0, totalImps-i)
		}

		imp2source[sourceId] = append(imp2source[sourceId], i)

	}

	totalReqs := len(imp2source)
	if 0 == totalReqs {
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	reqs := make([]*adapters.RequestData, 0, totalReqs)

	imps := request.Imp
	request.Imp = make([]openrtb.Imp, 0, len(imps))

	for sourceId, impIds := range imp2source {
		request.Imp = request.Imp[:0]

		for i := 0; i < len(impIds); i++ {
			request.Imp = append(request.Imp, imps[impIds[i]])
		}

		body, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, fmt.Errorf("error while encoding bidRequest, err: %s", err))
			return nil, errors
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri + fmt.Sprintf("?aid=%d", sourceId),
			Body:    body,
			Headers: headers,
		})
	}

	if 0 == len(reqs) {
		return nil, errors
	}

	return reqs, errors

}

func (a *AdtelligentAdapter) MakeBids(bidReq *openrtb.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) ([]*adapters.TypedBid, []error) {

	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpRes.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", httpRes.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{fmt.Errorf("error while decoding response, err: %s", err)}
	}

	var bids []*adapters.TypedBid
	var errors []error

	var impOK bool
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {

			impOK = false
			mediaType := openrtb_ext.BidTypeBanner
			for _, imp := range bidReq.Imp {
				if imp.ID == bid.ImpID {

					impOK = true

					if imp.Video != nil {
						mediaType = openrtb_ext.BidTypeVideo
						break
					}
				}
			}

			if !impOK {
				errors = append(errors, fmt.Errorf("ignoring bid id=%s, request doesn't contain any impression with id=%s", bid.ID, bid.ImpID))
				continue
			}

			bids = append(bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaType,
			})
		}
	}

	return bids, errors
}

func validateImpression(imp *openrtb.Imp) (int, error) {

	if imp.Banner == nil && imp.Video == nil {
		return 0, fmt.Errorf("ignoring imp id=%s, Adtelligent supports only Video and Banner", imp.ID)

	}

	if 0 == len(imp.Ext) {
		return 0, fmt.Errorf("ignoring imp id=%s, extImpBidder is empty", imp.ID)
	}

	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return 0, fmt.Errorf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err)
	}

	impExt := openrtb_ext.ExtImpAdtelligent{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return 0, fmt.Errorf("ignoring imp id=%s, error while decoding adtelligentImpExt, err: %s", imp.ID, err)
	}

	if impExt.SourceId == 0 {
		return 0, fmt.Errorf("ignoring imp id=%s, adtelligentImpExt doesn't contain the required field sourceId", imp.ID)
	}

	// common extension for all impressions
	var impExtBuffer []byte

	impExtBuffer, err = json.Marshal(&adtelligentImpExt{
		Adtelligent: impExt,
	})

	if impExt.BidFloor > 0 {
		imp.BidFloor = impExt.BidFloor
	}

	imp.Ext = impExtBuffer

	return impExt.SourceId, nil
}
func NewAdtelligentAdapter(config *adapters.HTTPAdapterConfig) *AdtelligentAdapter {
	return NewAdtelligentBidder(adapters.NewHTTPAdapter(config).Client)
}

func NewAdtelligentBidder(client *http.Client) *AdtelligentAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &AdtelligentAdapter{
		http: a,
		URI:  uri,
	}
}
