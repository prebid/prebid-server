package sspBC

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	adapterVersion              = "5.8"
	impFallbackSize             = "1x1"
	requestTypeStandard         = 1
	requestTypeOneCode          = 2
	requestTypeTest             = 3
	prebidServerIntegrationType = "4"
)

var (
	errSiteNill           = errors.New("site cannot be nill")
	errImpNotFound        = errors.New("imp not found")
	errNotSupportedFormat = errors.New("bid format is not supported")
)

// mcAd defines the MC payload for banner ads.
type mcAd struct {
	Id      string             `json:"id"`
	Seat    string             `json:"seat"`
	SeatBid []openrtb2.SeatBid `json:"seatbid"`
}

// adSlotData defines struct used for the oneCode detection.
type adSlotData struct {
	PbSlot string `json:"pbslot"`
	PbSize string `json:"pbsize"`
}

// templatePayload represents the banner template payload.
type templatePayload struct {
	SiteId  string `json:"siteid"`
	SlotId  string `json:"slotid"`
	AdLabel string `json:"adlabel"`
	PubId   string `json:"pubid"`
	Page    string `json:"page"`
	Referer string `json:"referer"`
	McAd    mcAd   `json:"mcad"`
	Inver   string `json:"inver"`
}

// requestImpExt represents the ext field of the request imp field.
type requestImpExt struct {
	Data adSlotData `json:"data"`
}

// responseExt represents ext data added by proxy.
type responseExt struct {
	AdLabel     string `json:"adlabel"`
	PublisherId string `json:"pubid"`
	SiteId      string `json:"siteid"`
	SlotId      string `json:"slotid"`
}

type adapter struct {
	endpoint       string
	bannerTemplate *template.Template
}

// ---------------ADAPTER INTERFACE------------------
// Builder builds a new instance of the sspBC adapter
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	// HTML template used to create banner ads
	const bannerHTML = `<html><head><title></title><meta charset="UTF-8"><meta name="viewport" content="` +
		`width=device-width, initial-scale=1.0"><style> body { background-color: transparent; margin: 0;` +
		` padding: 0; }</style><script> window.rekid = {{.SiteId}}; window.slot = {{.SlotId}}; window.ad` +
		`label = '{{.AdLabel}}'; window.pubid = '{{.PubId}}'; window.wp_sn = 'sspbc_go'; window.page = '` +
		`{{.Page}}'; window.ref = '{{.Referer}}'; window.mcad = {{.McAd}}; window.in` +
		`ver = '{{.Inver}}'; </script></head><body><div id="c"></div><script async c` +
		`rossorigin nomodule src="//std.wpcdn.pl/wpjslib/wpjslib-inline.js" id="wpjslib"></script><scrip` +
		`t async crossorigin type="module" src="//std.wpcdn.pl/wpjslib6/wpjslib-inline.js" id="wpjslib6"` +
		`></script></body></html>`

	bannerTemplate, err := template.New("banner").Parse(bannerHTML)
	if err != nil {
		return nil, err
	}

	bidder := &adapter{
		endpoint:       config.Endpoint,
		bannerTemplate: bannerTemplate,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	formattedRequest, err := formatSspBcRequest(request)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(formattedRequest)
	if err != nil {
		return nil, []error{err}
	}

	requestURL, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{err}
	}

	// add query parameters to request
	queryParams := requestURL.Query()
	queryParams.Add("bdver", adapterVersion)
	queryParams.Add("inver", prebidServerIntegrationType)
	requestURL.RawQuery = queryParams.Encode()

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    requestURL.String(),
		Body:   requestJSON,
		ImpIDs: getImpIDs(formattedRequest.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, externalResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if externalResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if externalResponse.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", externalResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(externalResponse.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	bidResponse.Currency = response.Cur

	var errors []error
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			if err := a.impToBid(internalRequest, seatBid, bid, bidResponse); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return bidResponse, errors
}

func (a *adapter) impToBid(internalRequest *openrtb2.BidRequest, seatBid openrtb2.SeatBid, bid openrtb2.Bid,
	bidResponse *adapters.BidderResponse) error {
	var bidType openrtb_ext.BidType

	/*
	  Determine bid type
	  At this moment we only check if bid contains Adm property

	  Later updates will check for video & native data
	*/
	if bid.AdM != "" {
		bidType = openrtb_ext.BidTypeBanner
	}

	/*
	  Recover original ImpID
	  (stored on request in TagID)
	*/
	impID, err := getOriginalImpID(bid.ImpID, internalRequest.Imp)
	if err != nil {
		return err
	}
	bid.ImpID = impID

	// read additional data from proxy
	var bidDataExt responseExt
	if err := jsonutil.Unmarshal(bid.Ext, &bidDataExt); err != nil {
		return err
	}
	/*
		use correct ad creation method for a detected bid type
		right now, we are only creating banner ads
		if type is not detected / supported, throw error
	*/
	if bidType != openrtb_ext.BidTypeBanner {
		return errNotSupportedFormat
	}

	var adCreationError error
	bid.AdM, adCreationError = a.createBannerAd(bid, bidDataExt, internalRequest, seatBid.Seat)
	if adCreationError != nil {
		return adCreationError
	}
	// append bid to responses
	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bid,
		BidType: bidType,
	})

	return nil
}

func getOriginalImpID(impID string, imps []openrtb2.Imp) (string, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			return imp.TagID, nil
		}
	}

	return "", errImpNotFound
}

func (a *adapter) createBannerAd(bid openrtb2.Bid, ext responseExt, request *openrtb2.BidRequest, seat string) (string, error) {
	if strings.Contains(bid.AdM, "<!--preformatted-->") {
		// Banner ad is already formatted
		return bid.AdM, nil
	}

	// create McAd payload
	var mcad = mcAd{
		Id:   request.ID,
		Seat: seat,
		SeatBid: []openrtb2.SeatBid{
			{Bid: []openrtb2.Bid{bid}},
		},
	}

	bannerData := &templatePayload{
		SiteId:  ext.SiteId,
		SlotId:  ext.SlotId,
		AdLabel: ext.AdLabel,
		PubId:   ext.PublisherId,
		Page:    request.Site.Page,
		Referer: request.Site.Ref,
		McAd:    mcad,
		Inver:   prebidServerIntegrationType,
	}

	var filledTemplate bytes.Buffer
	if err := a.bannerTemplate.Execute(&filledTemplate, bannerData); err != nil {
		return "", err
	}

	return filledTemplate.String(), nil
}

func getImpSize(imp openrtb2.Imp) string {
	if imp.Banner == nil || len(imp.Banner.Format) == 0 {
		return impFallbackSize
	}

	var (
		areaMax int64
		impSize = impFallbackSize
	)

	for _, size := range imp.Banner.Format {
		area := size.W * size.H
		if area > areaMax {
			areaMax = area
			impSize = fmt.Sprintf("%dx%d", size.W, size.H)
		}
	}

	return impSize
}

// getBidParameters reads additional data for this imp (site id , placement id, test)
// Errors in parameters do not break imp flow, and thus are not returned
func getBidParameters(imp openrtb2.Imp) openrtb_ext.ExtImpSspbc {
	var extBidder adapters.ExtImpBidder
	var extSSP openrtb_ext.ExtImpSspbc

	if err := jsonutil.Unmarshal(imp.Ext, &extBidder); err == nil {
		_ = jsonutil.Unmarshal(extBidder.Bidder, &extSSP)
	}

	return extSSP
}

// getRequestType checks what kind of request we have. It can either be:
// - a standard request, where all Imps have complete site / placement data
// - a oneCodeRequest, where site / placement data has to be determined by server
// - a test request, where server returns fixed example ads
func getRequestType(request *openrtb2.BidRequest) int {
	incompleteImps := 0

	for _, imp := range request.Imp {
		// Read data for this imp
		extSSP := getBidParameters(imp)

		if extSSP.IsTest != 0 {
			return requestTypeTest
		}

		if extSSP.SiteId == "" || extSSP.Id == "" {
			incompleteImps += 1
		}
	}

	if incompleteImps > 0 {
		return requestTypeOneCode
	}

	return requestTypeStandard
}

func formatSspBcRequest(request *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
	if request.Site == nil {
		return nil, errSiteNill
	}

	var siteID string

	// determine what kind of request we are dealing with
	requestType := getRequestType(request)

	for i, imp := range request.Imp {
		// read ext data for the impression
		extSSP := getBidParameters(imp)

		// store SiteID
		if extSSP.SiteId != "" {
			siteID = extSSP.SiteId
		}

		// save current imp.id (adUnit name) as imp.tagid
		// we will recover it in makeBids
		imp.TagID = imp.ID

		// if there is a placement id, and this is not a oneCodeRequest, use it in imp.id
		if extSSP.Id != "" && requestType != requestTypeOneCode {
			imp.ID = extSSP.Id
		}

		// check imp size and update e.ext - send pbslot, pbsize
		// inability to set bid.ext will cause request to be invalid
		impSize := getImpSize(imp)
		impExt := requestImpExt{
			Data: adSlotData{
				PbSlot: imp.TagID,
				PbSize: impSize,
			},
		}

		impExtJSON, err := json.Marshal(impExt)
		if err != nil {
			return nil, err
		}
		imp.Ext = impExtJSON
		// save updated imp
		request.Imp[i] = imp
	}

	siteCopy := *request.Site
	request.Site = &siteCopy

	/*
		update site ID
		for oneCode request it has to be blank
		for other requests it should be equal to
		SiteId from one of the bids
	*/
	if requestType == requestTypeOneCode || siteID == "" {
		request.Site.ID = ""
	} else {
		request.Site.ID = siteID
	}

	// add domain info
	if siteURL, err := url.Parse(request.Site.Page); err == nil {
		request.Site.Domain = siteURL.Hostname()
	}

	// set TEST Flag
	if requestType == requestTypeTest {
		request.Test = 1
	}

	return request, nil
}

// getImpIDs uses imp.TagID instead of imp.ID as formattedRequest stores imp.ID in imp.TagID
func getImpIDs(imps []openrtb2.Imp) []string {
	impIDs := make([]string, len(imps))
	for i := range imps {
		impIDs[i] = imps[i].TagID
	}
	return impIDs
}
