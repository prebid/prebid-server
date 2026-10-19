// Package optout implements the Opt Out Prebid Server bidder adapter.
package optout

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/prebid/go-gdpr/consentconstants"
	vendorconsent "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultCurrency        = "EUR"
	defaultTTL             = 300
	optoutVendorID  uint16 = 227
)

type adapter struct{ endpoint string }

type optoutSlot struct {
	AdSlot  string            `json:"adSlot"`
	ID      string            `json:"id"`
	Sizes   []adSize          `json:"sizes,omitempty"`
	Customs map[string]string `json:"customs,omitempty"`
}

type adSize struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type currencySettings struct {
	AdServerCurrency      string `json:"adServerCurrency"`
	GranularityMultiplier int    `json:"granularityMultiplier"`
}

type optoutRequest struct {
	Publisher  string            `json:"publisher"`
	Slots      []optoutSlot      `json:"slots"`
	Currency   currencySettings  `json:"cur"`
	URL        string            `json:"url"`
	SDKVersion string            `json:"sdk_version"`
	Consent    string            `json:"gdpr_consent"`
	GDPR       int               `json:"gdpr"`
	OS         string            `json:"os,omitempty"`
	Customs    map[string]string `json:"customs,omitempty"`
	ORTB2      string            `json:"ortb2,omitempty"`
}

type optoutBid struct {
	RequestID  string          `json:"requestId"`
	CPM        float64         `json:"cpm"`
	Currency   string          `json:"currency"`
	Width      int64           `json:"width"`
	Height     int64           `json:"height"`
	Ad         string          `json:"ad"`
	TTL        int64           `json:"ttl"`
	CreativeID string          `json:"creativeId"`
	OptOutExt  json.RawMessage `json:"optOutExt"`
	Meta       json.RawMessage `json:"meta"`
}

// Builder requires the single Opt Out DSP endpoint from the server configuration.
func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("optout: missing endpoint")
	}
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func decodeParams(imp openrtb2.Imp) (openrtb_ext.ImpExtOptout, error) {
	var ext adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return openrtb_ext.ImpExtOptout{}, fmt.Errorf("invalid impression extension: %w", err)
	}
	var p openrtb_ext.ImpExtOptout
	if len(ext.Bidder) == 0 {
		return p, fmt.Errorf("missing bidder parameters")
	}
	if err := jsonutil.Unmarshal(ext.Bidder, &p); err != nil {
		return p, fmt.Errorf("invalid bidder parameters: %w", err)
	}
	if p.Publisher == "" || p.AdSlot == "" {
		return p, fmt.Errorf("publisher and adSlot are required")
	}
	return p, nil
}

func normalizeCustoms(customs map[string]interface{}) map[string]string {
	result := map[string]string{}
	for k, v := range customs {
		if v == nil {
			continue
		}
		switch value := v.(type) {
		case []interface{}:
			vals := make([]string, len(value))
			for i, item := range value {
				if item != nil {
					vals[i] = fmt.Sprint(item)
				}
			}
			result[k] = strings.Join(vals, ",")
		case map[string]interface{}:
			if b, err := jsonutil.Marshal(value); err == nil {
				result[k] = string(b)
			}
		case string:
			result[k] = value
		default:
			result[k] = fmt.Sprint(value)
		}
	}
	return result
}

func mergeCustoms(dst map[string]string, data map[string]interface{}) {
	for k, v := range normalizeCustoms(data) {
		dst[k] = v
	}
}

func mergeExtData(dst map[string]string, ext json.RawMessage, source string) error {
	if len(ext) == 0 {
		return nil
	}
	var wrapped struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := jsonutil.Unmarshal(ext, &wrapped); err != nil {
		return &errortypes.BadInput{Message: fmt.Sprintf("optout: invalid %s.ext: %v", source, err)}
	}
	mergeCustoms(dst, wrapped.Data)
	return nil
}

func readConsent(req *openrtb2.BidRequest) string {
	if req == nil || req.User == nil {
		return ""
	}

	// OpenRTB 2.6: consent is directly in user.consent
	if req.User.Consent != "" {
		return req.User.Consent
	}

	// OpenRTB 2.5 and older: consent is in user.ext.consent
	var ext struct {
		Consent string `json:"consent"`
	}

	if err := jsonutil.Unmarshal(req.User.Ext, &ext); err != nil {
		return ""
	}

	return ext.Consent
}

// mayForwardPersonalData requires explicit TCF Purpose 2 and vendor consent
// when GDPR applies. Unknown GDPR applicability fails closed.
// This is an additional data-minimization rule, not a full TCF legal-basis check.
func mayForwardPersonalData(req *openrtb2.BidRequest, tcString string) bool {
	if req == nil || req.Regs == nil || req.Regs.GDPR == nil {
		return false
	}

	if *req.Regs.GDPR == 0 {
		return true // GDPR explicitly does not apply; respect other privacy controls upstream.
	}
	if tcString == "" {
		return false
	}
	consent, err := vendorconsent.ParseString(tcString)
	if err != nil || consent.Version() != 2 {
		return false
	}
	return consent.PurposeAllowed(consentconstants.Purpose(2)) &&
		consent.VendorConsent(optoutVendorID)
}

// makeForwardableORTB2 creates a separate OpenRTB request for forwarding.
// Never mutate the request shared by Prebid Server and other adapters.
// When the privacy check fails, replace the complete user object with a
// marker-only object and redact any IP addresses in device.
func makeForwardableORTB2(req *openrtb2.BidRequest, allowPersonalData bool) (*openrtb2.BidRequest, error) {
	raw, err := jsonutil.Marshal(req)
	if err != nil {
		return nil, err
	}
	var copied openrtb2.BidRequest
	if err := jsonutil.Unmarshal(raw, &copied); err != nil {
		return nil, err
	}
	if !allowPersonalData {
		copied.User = &openrtb2.User{
			Ext: json.RawMessage(`{"optout":{"redacted":true}}`),
		}
		if copied.Device != nil {
			if copied.Device.IP != "" {
				copied.Device.IP = ""
			}
			if copied.Device.IPv6 != "" {
				copied.Device.IPv6 = ""
			}
		}
	}
	return &copied, nil
}

func (a *adapter) MakeRequests(req *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if req == nil {
		return nil, []error{
			&errortypes.BadInput{Message: "optout: nil request"},
		}
	}

	var errors []error
	var publisher string
	var slots []optoutSlot
	var impIDs []string
	var includeOrtb2 bool

	for _, imp := range req.Imp {
		if imp.Banner == nil {
			errors = append(errors, &errortypes.BadInput{Message: fmt.Sprintf("optout: imp %s is not banner", imp.ID)})
			continue
		}

		p, err := decodeParams(imp)
		if err != nil {
			errors = append(errors, &errortypes.BadInput{Message: fmt.Sprintf("optout: imp %s: %v", imp.ID, err)})
			continue
		}

		if publisher == "" {
			publisher = p.Publisher
		} else if p.Publisher != publisher {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("optout: imp %s has publisher %s, expected %s", imp.ID, p.Publisher, publisher),
			})
			continue
		}

		slots = append(slots, optoutSlot{
			AdSlot:  p.AdSlot,
			ID:      imp.ID,
			Sizes:   getAllowedSizes(imp),
			Customs: normalizeCustoms(p.Customs),
		})
		impIDs = append(impIDs, imp.ID)
		includeOrtb2 = includeOrtb2 || p.IncludeOrtb2
	}

	if len(slots) == 0 {
		return nil, errors
	}

	gdpr := 1
	if req.Regs != nil && req.Regs.GDPR != nil && *req.Regs.GDPR == 0 {
		gdpr = 0
	}
	consent := readConsent(req)
	// Apply preliminary consent-based personal data filtering.
	forwardPersonalData := mayForwardPersonalData(req, consent)
	// Build a privacy-filtered clone for ortb2 and the top-level device field.
	forwardable, err := makeForwardableORTB2(req, forwardPersonalData)
	if err != nil {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("optout: cannot clone OpenRTB request: %v", err)}}
	}

	currency := defaultCurrency
	if len(req.Cur) > 0 {
		currency = req.Cur[0]
	}
	pageURL := ""
	if req.Site != nil {
		pageURL = req.Site.Page
	}
	// Shared first-party data belongs in request-level customs. Slot customs
	// are already attached independently to each slot above.
	requestCustoms := make(map[string]string)
	if req.Site != nil {
		if err := mergeExtData(requestCustoms, req.Site.Ext, "site"); err != nil {
			errors = append(errors, err)
		}
	}
	if req.App != nil {
		if err := mergeExtData(requestCustoms, req.App.Ext, "app"); err != nil {
			errors = append(errors, err)
		}
	}
	if forwardPersonalData && req.User != nil {
		if err := mergeExtData(requestCustoms, req.User.Ext, "user"); err != nil {
			errors = append(errors, err)
		}
	}

	data := optoutRequest{
		Publisher:  publisher,
		Slots:      slots,
		Currency:   currencySettings{AdServerCurrency: currency, GranularityMultiplier: 1},
		URL:        pageURL,
		SDKVersion: "prebid-server",
		Consent:    consent,
		GDPR:       gdpr,
		Customs:    requestCustoms,
	}
	if forwardable.Device != nil {
		data.OS = forwardable.Device.OS
	}
	if includeOrtb2 {
		ortbJSON, err := jsonutil.Marshal(forwardable)
		if err != nil {
			errors = append(errors, err)
		} else {
			data.ORTB2 = string(ortbJSON)
		}
	}

	body, err := jsonutil.Marshal(data)
	if err != nil {
		return nil, append(errors, err)
	}
	headers := http.Header{"Content-Type": []string{"application/json"}}
	if req.Device != nil {
		if req.Device.UA != "" {
			headers.Set("X-Device-User-Agent", req.Device.UA)
		}
		if forwardPersonalData {
			ip := req.Device.IP
			if ip == "" {
				ip = req.Device.IPv6
			}
			if ip != "" {
				headers.Set("X-Forwarded-For", ip)
			}
		}
	}

	result := []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  impIDs,
	}}

	return result, errors
}

func getAllowedSizes(imp openrtb2.Imp) []adSize {
	// Deduplicate banner sizes.
	seen := make(map[adSize]bool)
	sizes := make([]adSize, 0)
	if imp.Banner.W != nil && imp.Banner.H != nil {
		size := adSize{*imp.Banner.W, *imp.Banner.H}
		if !seen[size] {
			seen[size] = true
			sizes = append(sizes, size)
		}
	}
	for _, f := range imp.Banner.Format {
		size := adSize{f.W, f.H}
		if !seen[size] {
			seen[size] = true
			sizes = append(sizes, size)
		}
	}
	return sizes
}

// MakeBids validates the DSP response before converting it into OpenRTB bids.
func (a *adapter) MakeBids(req *openrtb2.BidRequest, outgoing *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response == nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "optout: nil response"}}
	}
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("optout: unexpected HTTP status %d", response.StatusCode)}}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("optout: unexpected HTTP status %d", response.StatusCode)}}
	}
	if outgoing == nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "optout: missing outgoing request"}}
	}

	var sent struct {
		Slots []optoutSlot `json:"slots"`
	}
	if err := jsonutil.Unmarshal(outgoing.Body, &sent); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("optout: invalid outgoing request: %v", err)}}
	}
	if len(sent.Slots) == 0 {
		return nil, []error{&errortypes.BadServerResponse{Message: "optout: outgoing request has no slots"}}
	}

	validImpIDs := make(map[string]bool, len(sent.Slots))
	for _, slot := range sent.Slots {
		validImpIDs[slot.ID] = true
	}

	// The DSP can return either a JSON array or an object containing "bids".
	var bids []optoutBid
	if err := jsonutil.Unmarshal(response.Body, &bids); err != nil {
		var wrapped struct {
			Bids []optoutBid `json:"bids"`
		}
		if err2 := jsonutil.Unmarshal(response.Body, &wrapped); err2 != nil {
			return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("optout: invalid response JSON: %v", err2)}}
		}
		if wrapped.Bids == nil {
			return nil, []error{&errortypes.BadServerResponse{Message: "optout: DSP response is missing bids array"}}
		}
		bids = wrapped.Bids
	}

	result := adapters.NewBidderResponse()
	currency := ""
	var errors []error
	for _, b := range bids {
		if !validImpIDs[b.RequestID] {
			errors = append(errors, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("optout: unknown impression ID %q", b.RequestID),
			})
			continue
		}

		if b.Currency == "" || b.Ad == "" || b.Width <= 0 || b.Height <= 0 ||
			b.CPM < 0 || math.IsNaN(b.CPM) || math.IsInf(b.CPM, 0) {
			errors = append(errors, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("optout: invalid bid for impression %q", b.RequestID),
			})
			continue
		}

		if currency != "" && currency != b.Currency {
			errors = append(errors, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("optout: mixed response currencies (%s and %s)", currency, b.Currency),
			})
			continue
		}

		var bidExt json.RawMessage
		if len(b.OptOutExt) > 0 && string(b.OptOutExt) != "null" {
			var ext map[string]json.RawMessage
			if err := jsonutil.Unmarshal(b.OptOutExt, &ext); err != nil || ext == nil {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("optout: invalid bid extension for impression %q", b.RequestID),
				})
				continue
			}
			bidExt = b.OptOutExt
		}

		ttl := b.TTL
		if ttl <= 0 {
			ttl = defaultTTL
		}

		bid := &openrtb2.Bid{
			ID:    b.RequestID,
			ImpID: b.RequestID,
			Price: b.CPM,
			AdM:   b.Ad,
			W:     b.Width,
			H:     b.Height,
			CrID:  b.CreativeID,
			Exp:   ttl,
			Ext:   bidExt,
		}
		currency = b.Currency
		result.Bids = append(result.Bids, &adapters.TypedBid{
			Bid: bid, BidType: openrtb_ext.BidTypeBanner,
		})
	}
	if currency == "" {
		currency = defaultCurrency
	}
	result.Currency = currency

	return result, errors
}
