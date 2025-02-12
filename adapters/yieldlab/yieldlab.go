package yieldlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"golang.org/x/text/currency"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

// YieldlabAdapter connects the Yieldlab API to prebid server
type YieldlabAdapter struct {
	endpoint    string
	cacheBuster cacheBuster
	getWeek     weekGenerator
}

// Builder builds a new instance of the Yieldlab adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &YieldlabAdapter{
		endpoint:    config.Endpoint,
		cacheBuster: defaultCacheBuster,
		getWeek:     defaultWeekGenerator,
	}
	return bidder, nil
}

// makeEndpointURL builds endpoint url based on adapter-specific pub settings from imp.ext
func (a *YieldlabAdapter) makeEndpointURL(req *openrtb2.BidRequest, params *openrtb_ext.ExtImpYieldlab) (string, error) {
	uri, err := url.Parse(a.endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse yieldlab endpoint: %v", err)
	}

	uri.Path = path.Join(uri.Path, params.AdslotID)
	q := uri.Query()
	q.Set("content", "json")
	q.Set("pvid", "true")
	q.Set("ts", a.cacheBuster())
	q.Set("t", a.makeTargetingValues(params))

	if hasFormats, formats := a.makeFormats(req); hasFormats {
		q.Set("sizes", formats)
	}

	if req.User != nil && req.User.BuyerUID != "" {
		q.Set("ids", "ylid:"+req.User.BuyerUID)
	}

	if req.Device != nil {
		q.Set("yl_rtb_ifa", req.Device.IFA)
		q.Set("yl_rtb_devicetype", fmt.Sprintf("%v", req.Device.DeviceType))

		if req.Device.ConnectionType != nil {
			q.Set("yl_rtb_connectiontype", fmt.Sprintf("%v", req.Device.ConnectionType.Val()))
		}

		if req.Device.Geo != nil {
			q.Set("lat", fmt.Sprintf("%v", ptrutil.ValueOrDefault(req.Device.Geo.Lat)))
			q.Set("lon", fmt.Sprintf("%v", ptrutil.ValueOrDefault(req.Device.Geo.Lon)))
		}
	}

	if req.App != nil {
		q.Set("pubappname", req.App.Name)
		q.Set("pubbundlename", req.App.Bundle)
	}

	gdpr, consent, err := a.getGDPR(req)
	if err != nil {
		return "", err
	}
	if gdpr != "" {
		q.Set("gdpr", gdpr)
	}
	if consent != "" {
		q.Set("gdpr_consent", consent)
	}

	if req.Source != nil && req.Source.Ext != nil {
		if openRtbSchain := unmarshalSupplyChain(req); openRtbSchain != nil {
			if schainValue := makeSupplyChain(*openRtbSchain); schainValue != "" {
				q.Set("schain", schainValue)
			}
		}
	}

	dsa, err := getDSA(req)
	if err != nil {
		return "", err
	}
	if dsa != nil {
		if dsa.Required != nil {
			q.Set("dsarequired", strconv.Itoa(*dsa.Required))
		}
		if dsa.PubRender != nil {
			q.Set("dsapubrender", strconv.Itoa(*dsa.PubRender))
		}
		if dsa.DataToPub != nil {
			q.Set("dsadatatopub", strconv.Itoa(*dsa.DataToPub))
		}
		if len(dsa.Transparency) != 0 {
			transparencyParam := makeDSATransparencyURLParam(dsa.Transparency)
			if len(transparencyParam) != 0 {
				q.Set("dsatransparency", transparencyParam)
			}
		}
	}

	uri.RawQuery = q.Encode()

	return uri.String(), nil
}

// getDSA extracts the Digital Service Act (DSA) properties from the request.
func getDSA(req *openrtb2.BidRequest) (*dsaRequest, error) {
	if req.Regs == nil || req.Regs.Ext == nil {
		return nil, nil
	}

	var extRegs openRTBExtRegsWithDSA
	err := jsonutil.Unmarshal(req.Regs.Ext, &extRegs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Regs.Ext object from Yieldlab response: %v", err)
	}

	return extRegs.DSA, nil
}

// makeDSATransparencyURLParam creates the transparency url parameter
// as specified by the OpenRTB 2.X DSA Transparency community extension.
//
// Example result: platform1domain.com~1~~SSP2domain.com~1_2
func makeDSATransparencyURLParam(transparencyObjects []dsaTransparency) string {
	valueSeparator, itemSeparator, objectSeparator := "_", "~", "~~"

	var b strings.Builder

	concatParams := func(params []int) {
		b.WriteString(strconv.Itoa(params[0]))
		for _, param := range params[1:] {
			b.WriteString(valueSeparator)
			b.WriteString(strconv.Itoa(param))
		}
	}

	concatTransparency := func(object dsaTransparency) {
		if len(object.Domain) == 0 {
			return
		}

		b.WriteString(object.Domain)
		if len(object.Params) != 0 {
			b.WriteString(itemSeparator)
			concatParams(object.Params)
		}
	}

	concatTransparencies := func(objects []dsaTransparency) {
		if len(objects) == 0 {
			return
		}

		concatTransparency(objects[0])
		for _, obj := range objects[1:] {
			b.WriteString(objectSeparator)
			concatTransparency(obj)
		}
	}

	concatTransparencies(transparencyObjects)

	return b.String()
}

func (a *YieldlabAdapter) makeFormats(req *openrtb2.BidRequest) (bool, string) {
	var formats []string
	const sizesSeparator, adslotSizesSeparator = "|", ","
	for _, impression := range req.Imp {
		if !impIsTypeBannerOnly(impression) {
			continue
		}

		var formatsPerAdslot []string
		for _, format := range impression.Banner.Format {
			formatsPerAdslot = append(formatsPerAdslot, fmt.Sprintf("%dx%d", format.W, format.H))
		}
		adslotID := a.extractAdslotID(impression)
		sizesForAdslot := strings.Join(formatsPerAdslot, sizesSeparator)
		formats = append(formats, fmt.Sprintf("%s:%s", adslotID, sizesForAdslot))
	}
	return len(formats) != 0, strings.Join(formats, adslotSizesSeparator)
}

func (a *YieldlabAdapter) getGDPR(request *openrtb2.BidRequest) (string, string, error) {
	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtUser in Yieldlab GDPR check: %v", err)
		}
		consent = extUser.Consent
	}

	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
			if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
				gdpr = strconv.Itoa(int(*extRegs.GDPR))
			}
		}
	}

	return gdpr, consent, nil
}

func (a *YieldlabAdapter) makeTargetingValues(params *openrtb_ext.ExtImpYieldlab) string {
	values := url.Values{}
	for k, v := range params.Targeting {
		values.Set(k, v)
	}
	return values.Encode()
}

func (a *YieldlabAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{fmt.Errorf("invalid request %+v, no Impressions given", request)}
	}

	bidURL, err := a.makeEndpointURL(request, a.mergeParams(a.parseRequest(request)))
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Accept", "application/json")
	if request.Site != nil {
		headers.Add("Referer", request.Site.Page)
	}
	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)
		headers.Add("X-Forwarded-For", request.Device.IP)
	}
	if request.User != nil {
		headers.Add("Cookie", "id="+request.User.BuyerUID)
	}

	return []*adapters.RequestData{{
		Method:  "GET",
		Uri:     bidURL,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

// parseRequest extracts the Yieldlab request information from the request
func (a *YieldlabAdapter) parseRequest(request *openrtb2.BidRequest) []*openrtb_ext.ExtImpYieldlab {
	params := make([]*openrtb_ext.ExtImpYieldlab, 0)

	for i := 0; i < len(request.Imp); i++ {
		bidderExt := new(adapters.ExtImpBidder)
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, bidderExt); err != nil {
			continue
		}

		yieldlabExt := new(openrtb_ext.ExtImpYieldlab)
		if err := jsonutil.Unmarshal(bidderExt.Bidder, yieldlabExt); err != nil {
			continue
		}

		params = append(params, yieldlabExt)
	}

	return params
}

func (a *YieldlabAdapter) mergeParams(params []*openrtb_ext.ExtImpYieldlab) *openrtb_ext.ExtImpYieldlab {
	var adSlotIds []string
	targeting := make(map[string]string)

	for _, p := range params {
		adSlotIds = append(adSlotIds, p.AdslotID)
		for k, v := range p.Targeting {
			targeting[k] = v
		}
	}

	return &openrtb_ext.ExtImpYieldlab{
		AdslotID:  strings.Join(adSlotIds, adSlotIdSeparator),
		Targeting: targeting,
	}
}

// MakeBids make the bids for the bid response.
func (a *YieldlabAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode != 200 {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("failed to resolve bids from yieldlab response: Unexpected response code %v", response.StatusCode),
			},
		}
	}

	bids := make([]*bidResponse, 0)
	if err := jsonutil.Unmarshal(response.Body, &bids); err != nil {
		return nil, []error{
			&errortypes.BadServerResponse{
				Message: fmt.Sprintf("failed to parse bids response from yieldlab: %v", err),
			},
		}
	}

	params := a.parseRequest(internalRequest)

	bidderResponse := &adapters.BidderResponse{
		Currency: currency.EUR.String(),
		Bids:     []*adapters.TypedBid{},
	}

	adslotToImpMap := make(map[string]*openrtb2.Imp)
	for i := 0; i < len(internalRequest.Imp); i++ {
		adslotID := a.extractAdslotID(internalRequest.Imp[i])
		if internalRequest.Imp[i].Video != nil || internalRequest.Imp[i].Banner != nil {
			adslotToImpMap[adslotID] = &internalRequest.Imp[i]
		}
	}

	var bidErrors []error
	for _, bid := range bids {
		width, height, err := splitSize(bid.Adsize)
		if err != nil {
			return nil, []error{err}
		}

		req := a.findBidReq(bid.ID, params)
		if req == nil {
			return nil, []error{
				fmt.Errorf("failed to find yieldlab request for adslotID %v. This is most likely a programming issue", bid.ID),
			}
		}

		if imp, exists := adslotToImpMap[strconv.FormatUint(bid.ID, 10)]; !exists {
			continue
		} else {
			extJson, err := makeResponseExt(bid)
			if err != nil {
				bidErrors = append(bidErrors, err)
				// skip as bids with missing ext.dsa will be discarded anyway
				continue
			}

			responseBid := &openrtb2.Bid{
				ID:     strconv.FormatUint(bid.ID, 10),
				Price:  float64(bid.Price) / 100,
				ImpID:  imp.ID,
				CrID:   a.makeCreativeID(req, bid),
				DealID: strconv.FormatUint(bid.Pid, 10),
				W:      int64(width),
				H:      int64(height),
				Ext:    extJson,
			}

			var bidType openrtb_ext.BidType
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
				responseBid.NURL = a.makeAdSourceURL(internalRequest, req, bid)
				responseBid.AdM = a.makeVast(internalRequest, req, bid)
			} else if imp.Banner != nil {
				bidType = openrtb_ext.BidTypeBanner
				responseBid.AdM = a.makeBannerAdSource(internalRequest, req, bid)
			} else {
				// Yieldlab adapter currently doesn't support Audio and Native ads
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				BidType: bidType,
				Bid:     responseBid,
			})
		}
	}

	return bidderResponse, bidErrors
}

func makeResponseExt(bid *bidResponse) (json.RawMessage, error) {
	if bid.DSA != nil {
		extJson, err := json.Marshal(responseExtWithDSA{*bid.DSA})
		if err != nil {
			return nil, fmt.Errorf("failed to make JSON for seatbid.bid.ext for adslotID %v. This is most likely a programming issue", bid.ID)
		}
		return extJson, nil
	}
	return nil, nil
}

func (a *YieldlabAdapter) findBidReq(adslotID uint64, params []*openrtb_ext.ExtImpYieldlab) *openrtb_ext.ExtImpYieldlab {
	slotIdStr := strconv.FormatUint(adslotID, 10)

	for _, p := range params {
		if p.AdslotID == slotIdStr {
			return p
		}
	}

	return nil
}

func (a *YieldlabAdapter) extractAdslotID(internalRequestImp openrtb2.Imp) string {
	bidderExt := new(adapters.ExtImpBidder)
	jsonutil.Unmarshal(internalRequestImp.Ext, bidderExt)
	yieldlabExt := new(openrtb_ext.ExtImpYieldlab)
	jsonutil.Unmarshal(bidderExt.Bidder, yieldlabExt)
	return yieldlabExt.AdslotID
}

func (a *YieldlabAdapter) makeBannerAdSource(req *openrtb2.BidRequest, ext *openrtb_ext.ExtImpYieldlab, res *bidResponse) string {
	return fmt.Sprintf(adSourceBanner, a.makeAdSourceURL(req, ext, res))
}

func (a *YieldlabAdapter) makeVast(req *openrtb2.BidRequest, ext *openrtb_ext.ExtImpYieldlab, res *bidResponse) string {
	return fmt.Sprintf(vastMarkup, ext.AdslotID, a.makeAdSourceURL(req, ext, res))
}

func (a *YieldlabAdapter) makeAdSourceURL(req *openrtb2.BidRequest, ext *openrtb_ext.ExtImpYieldlab, res *bidResponse) string {
	val := url.Values{}
	val.Set("ts", a.cacheBuster())
	val.Set("id", ext.ExtId)
	val.Set("pvid", res.Pvid)

	if req.User != nil {
		val.Set("ids", "ylid:"+req.User.BuyerUID)
	}

	gdpr, consent, err := a.getGDPR(req)
	if err == nil && gdpr != "" && consent != "" {
		val.Set("gdpr", gdpr)
		val.Set("gdpr_consent", consent)
	}

	return fmt.Sprintf(adSourceURL, ext.AdslotID, ext.SupplyID, res.Adsize, val.Encode())
}

func (a *YieldlabAdapter) makeCreativeID(req *openrtb_ext.ExtImpYieldlab, bid *bidResponse) string {
	return fmt.Sprintf(creativeID, req.AdslotID, bid.Pid, a.getWeek())
}

// unmarshalSupplyChain makes the value for the schain URL parameter from the openRTB schain object.
func unmarshalSupplyChain(req *openrtb2.BidRequest) *openrtb2.SupplyChain {
	var extSChain openrtb_ext.ExtRequestPrebidSChain
	err := jsonutil.Unmarshal(req.Source.Ext, &extSChain)
	if err != nil {
		// req.Source.Ext could be anything so don't handle any errors
		return nil
	}
	return &extSChain.SChain
}

// makeNodeValue makes the value for the schain URL parameter from the openRTB schain object.
func makeSupplyChain(openRtbSchain openrtb2.SupplyChain) string {
	if len(openRtbSchain.Nodes) == 0 {
		return ""
	}

	const schainPrefixFmt = "%s,%d"
	const schainNodeFmt = "!%s,%s,%s,%s,%s,%s,%s"
	schainPrefix := fmt.Sprintf(schainPrefixFmt, openRtbSchain.Ver, openRtbSchain.Complete)
	var sb strings.Builder
	sb.WriteString(schainPrefix)
	for _, node := range openRtbSchain.Nodes {
		// has to be in order: asi,sid,hp,rid,name,domain,ext
		schainNode := fmt.Sprintf(
			schainNodeFmt,
			makeNodeValue(node.ASI),
			makeNodeValue(node.SID),
			makeNodeValue(node.HP),
			makeNodeValue(node.RID),
			makeNodeValue(node.Name),
			makeNodeValue(node.Domain),
			makeNodeValue(node.Ext),
		)
		sb.WriteString(schainNode)
	}
	return sb.String()
}

// makeNodeValue converts any known value type from a schain node to a string and does URL encoding if necessary.
func makeNodeValue(nodeParam any) string {
	switch nodeParam := nodeParam.(type) {
	case string:
		return url.QueryEscape(nodeParam)
	case *int8:
		pointer := nodeParam
		if pointer == nil {
			return ""
		}
		return makeNodeValue(int(*pointer))
	case int:
		return strconv.Itoa(nodeParam)
	case json.RawMessage:
		if nodeParam != nil {
			freeFormJson, err := json.Marshal(nodeParam)
			if err != nil {
				return ""
			}
			return makeNodeValue(string(freeFormJson))
		}
		return ""
	default:
		return ""
	}
}

func splitSize(size string) (uint64, uint64, error) {
	sizeParts := strings.Split(size, adsizeSeparator)
	if len(sizeParts) != 2 {
		return 0, 0, nil
	}

	width, err := strconv.ParseUint(sizeParts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse yieldlab adsize: %v", err)
	}

	height, err := strconv.ParseUint(sizeParts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse yieldlab adsize: %v", err)
	}

	return width, height, nil

}

// impIsTypeBannerOnly returns true if impression is only from type banner. Mixed typed with banner would also result in false.
func impIsTypeBannerOnly(impression openrtb2.Imp) bool {
	return impression.Banner != nil && impression.Audio == nil && impression.Video == nil && impression.Native == nil
}
