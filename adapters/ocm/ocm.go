package ocm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// mTypeUnset is the zero value of openrtb2.MarkupType, meaning the response did
// not state a media type. OpenRTB defines no 0 markup type, so it is only ever an
// omission.
const mTypeUnset openrtb2.MarkupType = 0

// adapter routes bid requests to the Orange Click Media exchange.
type adapter struct {
	endpoint string
}

// Builder builds a new instance of the OCM adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

// MakeRequests packs every impression into a single OpenRTB request for the OCM
// exchange.
//
// The publisherId param identifies the seller and is written to the request-level
// publisher ID, so all impressions in one request must agree on it; a divergent
// publisherId is a misconfiguration and rejects the whole request.
//
// The placementId param names the impression-level stored request that the OCM
// endpoint expands into the impression's real configuration, so it is encoded as
// imp.ext.prebid.storedrequest.id rather than as a plain slot identifier. It is
// required: an impression referencing no stored request cannot be resolved
// downstream, and would fail the whole request rather than just itself.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, nil
	}

	var publisherID string
	for i := range request.Imp {
		impExt, err := parseImpExt(&request.Imp[i])
		if err != nil {
			return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: %s", i, err)}}
		}

		if i == 0 {
			publisherID = impExt.PublisherID
		} else if impExt.PublisherID != publisherID {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("imp[%d]: all impressions must share the same publisherId, found %q and %q", i, impExt.PublisherID, publisherID),
			}}
		}

		// Imp elements are safe to modify in place: prebid-server hands each
		// adapter its own shallow copy of the imp slice.
		if err := setStoredRequestID(&request.Imp[i], impExt.PlacementID); err != nil {
			return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("imp[%d]: %s", i, err)}}
		}
	}

	if err := setPublisherID(request, publisherID); err != nil {
		return nil, []error{&errortypes.BadInput{Message: err.Error()}}
	}
	if err := removeOCMAliases(request); err != nil {
		return nil, []error{&errortypes.BadInput{Message: err.Error()}}
	}

	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{fmt.Errorf("unable to marshal bid request: %w", err)}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

// parseImpExt decodes imp.ext.bidder into ExtImpOcm and validates it. Both params
// are required and must be non-blank; the JSON schema already enforces presence,
// and the whitespace check here covers values the schema accepts but the OCM
// endpoint cannot resolve.
func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpOcm, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, fmt.Errorf("unable to unmarshal ext: %s", err)
	}

	var impExt openrtb_ext.ExtImpOcm
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, fmt.Errorf("unable to unmarshal ext.bidder: %s", err)
	}

	if strings.TrimSpace(impExt.PublisherID) == "" {
		return nil, errors.New("publisherId is required and must not be blank")
	}
	if strings.TrimSpace(impExt.PlacementID) == "" {
		return nil, errors.New("placementId is required and must not be blank")
	}

	return &impExt, nil
}

// setStoredRequestID consumes imp.ext.bidder and records placementId as the
// impression's stored-request ID at imp.ext.prebid.storedrequest.id, which is
// where the OCM endpoint looks for it.
//
// The surrounding ext is rebuilt through a generic map rather than a typed struct
// so that keys this adapter knows nothing about — gpid, tid, data and any future
// additions — survive the round trip untouched.
func setStoredRequestID(imp *openrtb2.Imp, placementID string) error {
	ext := map[string]json.RawMessage{}
	if len(imp.Ext) > 0 {
		if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
			return fmt.Errorf("unable to unmarshal ext: %s", err)
		}
		if ext == nil {
			ext = map[string]json.RawMessage{}
		}
	}
	delete(ext, "bidder")

	prebid := map[string]json.RawMessage{}
	if raw, exists := ext["prebid"]; exists && len(raw) > 0 {
		if err := jsonutil.Unmarshal(raw, &prebid); err != nil {
			return fmt.Errorf("unable to unmarshal ext.prebid: %s", err)
		}
		if prebid == nil {
			prebid = map[string]json.RawMessage{}
		}
	}

	storedRequest, err := jsonutil.Marshal(openrtb_ext.ExtStoredRequest{ID: placementID})
	if err != nil {
		return err
	}
	prebid["storedrequest"] = storedRequest

	prebidJSON, err := jsonutil.Marshal(prebid)
	if err != nil {
		return err
	}
	ext["prebid"] = prebidJSON

	extJSON, err := jsonutil.Marshal(ext)
	if err != nil {
		return err
	}

	imp.Ext = extJSON
	return nil
}

// setPublisherID writes publisherId to the request-level publisher ID and removes
// the originating host's parent account. Site, App, Publisher and publisher.ext
// are copied before mutation because prebid-server shares them across adapters.
func setPublisherID(request *openrtb2.BidRequest, publisherID string) error {
	if request.Site != nil {
		site := *request.Site
		publisher, err := publisherWithID(site.Publisher, publisherID)
		if err != nil {
			return fmt.Errorf("unable to update site.publisher: %w", err)
		}
		site.Publisher = publisher
		request.Site = &site
		return nil
	}

	if request.App != nil {
		app := *request.App
		publisher, err := publisherWithID(app.Publisher, publisherID)
		if err != nil {
			return fmt.Errorf("unable to update app.publisher: %w", err)
		}
		app.Publisher = publisher
		request.App = &app
	}
	return nil
}

// publisherWithID returns a copy of publisher carrying id and without
// ext.prebid.parentAccount, or a fresh Publisher when the request omitted one.
func publisherWithID(publisher *openrtb2.Publisher, id string) (*openrtb2.Publisher, error) {
	if publisher == nil {
		return &openrtb2.Publisher{ID: id}, nil
	}

	cpy := *publisher
	cpy.ID = id

	ext, err := removeParentAccount(publisher.Ext)
	if err != nil {
		return nil, err
	}
	cpy.Ext = ext
	return &cpy, nil
}

// removeParentAccount removes only publisher.ext.prebid.parentAccount, retaining
// every other publisher extension field.
func removeParentAccount(publisherExt json.RawMessage) (json.RawMessage, error) {
	if len(publisherExt) == 0 {
		return publisherExt, nil
	}

	ext := map[string]json.RawMessage{}
	if err := jsonutil.Unmarshal(publisherExt, &ext); err != nil {
		return nil, fmt.Errorf("unable to unmarshal ext: %s", err)
	}

	prebidJSON, exists := ext[openrtb_ext.PrebidExtKey]
	if !exists || len(prebidJSON) == 0 {
		return publisherExt, nil
	}

	prebid := map[string]json.RawMessage{}
	if err := jsonutil.Unmarshal(prebidJSON, &prebid); err != nil {
		return nil, fmt.Errorf("unable to unmarshal ext.%s: %s", openrtb_ext.PrebidExtKey, err)
	}
	if _, exists := prebid["parentAccount"]; !exists {
		return publisherExt, nil
	}

	delete(prebid, "parentAccount")
	prebidJSON, err := jsonutil.Marshal(prebid)
	if err != nil {
		return nil, err
	}
	ext[openrtb_ext.PrebidExtKey] = prebidJSON

	return jsonutil.Marshal(ext)
}

// removeOCMAliases removes aliases that resolve to this adapter before forwarding
// the request to another Prebid Server. Other aliases and request extensions are
// retained, and request.Ext receives new backing storage rather than mutating the
// shared input bytes.
func removeOCMAliases(request *openrtb2.BidRequest) error {
	if len(request.Ext) == 0 {
		return nil
	}

	ext := map[string]json.RawMessage{}
	if err := jsonutil.Unmarshal(request.Ext, &ext); err != nil {
		return fmt.Errorf("unable to unmarshal request.ext: %s", err)
	}

	prebidJSON, exists := ext[openrtb_ext.PrebidExtKey]
	if !exists || len(prebidJSON) == 0 {
		return nil
	}
	prebid := map[string]json.RawMessage{}
	if err := jsonutil.Unmarshal(prebidJSON, &prebid); err != nil {
		return fmt.Errorf("unable to unmarshal request.ext.%s: %s", openrtb_ext.PrebidExtKey, err)
	}

	aliasesJSON, exists := prebid["aliases"]
	if !exists || len(aliasesJSON) == 0 {
		return nil
	}
	aliases := map[string]string{}
	if err := jsonutil.Unmarshal(aliasesJSON, &aliases); err != nil {
		return fmt.Errorf("unable to unmarshal request.ext.%s.aliases: %s", openrtb_ext.PrebidExtKey, err)
	}

	changed := false
	for alias, bidder := range aliases {
		if bidder == string(openrtb_ext.BidderOcm) {
			delete(aliases, alias)
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if len(aliases) == 0 {
		delete(prebid, "aliases")
	} else {
		aliasesJSON, err := jsonutil.Marshal(aliases)
		if err != nil {
			return err
		}
		prebid["aliases"] = aliasesJSON
	}
	prebidJSON, err := jsonutil.Marshal(prebid)
	if err != nil {
		return err
	}
	ext[openrtb_ext.PrebidExtKey] = prebidJSON

	requestExt, err := jsonutil.Marshal(ext)
	if err != nil {
		return err
	}
	request.Ext = requestExt
	return nil
}

// MakeBids unpacks the OCM exchange's OpenRTB bid response. Bids whose media
// type cannot be resolved are skipped and reported, so one malformed bid does
// not discard the rest of the seat.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &seatBid.Bid[i],
				BidType:  bidType,
				BidVideo: videoFromBidExt(seatBid.Bid[i].Ext, bidType),
			})
		}
	}

	return bidResponse, errs
}

// videoFromBidExt returns targeting metadata supplied by the receiving PBS for
// video bids. Invalid, unrelated, and non-video bid extensions are ignored.
func videoFromBidExt(bidExt json.RawMessage, bidType openrtb_ext.BidType) *openrtb_ext.ExtBidPrebidVideo {
	if bidType != openrtb_ext.BidTypeVideo || len(bidExt) == 0 {
		return nil
	}

	var ext openrtb_ext.ExtBid
	if err := jsonutil.Unmarshal(bidExt, &ext); err != nil || ext.Prebid == nil {
		return nil
	}
	return ext.Prebid.Video
}

// getMediaTypeForBid resolves the bid's media type from the OpenRTB 2.6 bid.mtype
// field, which is authoritative whenever it is set.
//
// Not every upstream populates mtype: a Prebid Server passes through whatever
// mtype its winning adapter set, and reports the type it resolved in
// bid.ext.prebid.type instead (see exchange/entities.PbsOrtbBid). Treating an
// absent mtype as fatal would therefore discard perfectly good bids, so
// ext.prebid.type is consulted in that one case.
//
// A set-but-unsupported mtype (audio, or a value this adapter does not know) is
// rejected outright and never allowed to fall through to the extension. Doing
// otherwise would let an ext of "banner" reclassify an audio bid, contradicting
// the authoritative field and delivering a media type OCM does not declare.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case mTypeUnset:
		if bidType, ok := mediaTypeFromBidExt(bid.Ext); ok {
			return bidType, nil
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unable to determine media type for bid %q: no mtype and no ext.prebid.type", bid.ID),
		}
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %q", bid.MType, bid.ID),
		}
	}
}

// mediaTypeFromBidExt reads a media type from bid.ext.prebid.type, accepting
// only the types OCM declares support for. A malformed or absent ext is not an
// error here; the caller reports the failure to resolve a type.
func mediaTypeFromBidExt(bidExt json.RawMessage) (openrtb_ext.BidType, bool) {
	if len(bidExt) == 0 {
		return "", false
	}

	var parsed struct {
		Prebid struct {
			Type openrtb_ext.BidType `json:"type"`
		} `json:"prebid"`
	}
	if err := jsonutil.Unmarshal(bidExt, &parsed); err != nil {
		return "", false
	}

	switch parsed.Prebid.Type {
	case openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo, openrtb_ext.BidTypeNative:
		return parsed.Prebid.Type, true
	}

	return "", false
}
