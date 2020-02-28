package parser

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics/pubstack/model"
	"github.com/ua-parser/uap-go/uaparser"
	"strings"
)

type UserAgentParser struct {
	p *uaparser.Parser
}

type Parser struct {
	scope string
	uap   *UserAgentParser
}

type AuctionMetadata struct {
	device         string
	country        string
	domain         string
	href           string
	auctionID      string
	browserVersion string
	browserName    string
	osVersion      string
	osName         string
	refresh        bool
}

func NewParser(scope string) *Parser {
	fmt.Printf("[PBSTCK] initializing listener for scope %s", scope)
	return &Parser{
		scope: scope,
		uap:   &UserAgentParser{uaparser.NewFromSaved()},
	}
}

func (p *Parser) Feed(req *openrtb.BidRequest, resp *openrtb.BidResponse) []model.Auction {
	mapBidAuction := make(map[string]model.Auction)

	// Retrieve auctions metas
	auctionMetas := p.parseAuctionMetadata(req)

	// Retrieve per auctions infos
	for _, impRequest := range req.Imp {
		sizes := p.parseRequestedSizes(impRequest)

		auction := p.buildAuction(auctionMetas, impRequest.ID, sizes)
		if _, ok := mapBidAuction[impRequest.ID]; ok {
			fmt.Println("auction already filled")
		} else {
			mapBidAuction[impRequest.ID] = auction
		}
	}
	final := make([]model.Auction, 0)

	bidsMap := p.parseBids(resp)

	for impid, bids := range bidsMap {
		for aucImpid, auction := range mapBidAuction {
			if aucImpid == impid {
				auction.BidRequests = bids
				mapBidAuction[aucImpid] = auction
			}
		}
	}

	for _, a := range mapBidAuction {
		final = append(final, a)
	}

	return final
}

func (p *Parser) parseBids(resp *openrtb.BidResponse) map[string][]model.BidRequest {

	result := make(map[string][]model.BidRequest)

	for _, bidder := range resp.SeatBid {
		bidderCode := getBidderCode(bidder)
		for _, bid := range bidder.Bid {
			if _, ok := result[bid.ImpID]; !ok {
				result[bid.ImpID] = make([]model.BidRequest, 0)
			}
			result[bid.ImpID] = append(result[bid.ImpID], translateBid(bid, bidderCode))
		}
	}
	return result
}

func translateBid(bid openrtb.Bid, code string) model.BidRequest {
	state := "noBid"
	if bid.Price > 0.0 {
		state = "bid"
	}

	return model.BidRequest{
		BidderCode: code,
		State:      state,
		Cpm:        float32(bid.Price),
		Elapsed:    0,
		BidId:      bid.ID,
		Tags:       nil,
	}
}

func getBidderCode(seat openrtb.SeatBid) string {
	if seat.Seat == "improvedigital" {
		return seat.Seat
	}

	extRaw := make(map[string]interface{})

	_ = json.Unmarshal(seat.Ext, &extRaw)
	for key := range extRaw {
		// this is not working
		if key == "appnexus" {
			return key
		}
	}
	return "N/A"
}

// Retrieve requested sizes for an auction request
func (p *Parser) parseRequestedSizes(impRequest openrtb.Imp) []string {
	requestedSizes := make([]string, 0)

	// only support banner at the moment
	if impRequest.Banner != nil {
		if len(impRequest.Banner.Format) == 0 {
			requestedSizes = append(requestedSizes, fmt.Sprintf("%dx%d", *impRequest.Banner.W, *impRequest.Banner.H))
		} else {
			for _, format := range impRequest.Banner.Format {
				requestedSizes = append(requestedSizes, fmt.Sprintf("%dx%d", format.W, format.H))
			}
		}
	}
	return requestedSizes
}

// Retrieve infos related to the top level objects and applicable to all auctions
func (p *Parser) parseAuctionMetadata(req *openrtb.BidRequest) *AuctionMetadata {

	bwFamily := ""
	bwVersion := ""
	osFamily := ""
	osVersion := ""

	// parse UA -> req.Device.UA
	if req.Device != nil {
		parsedUA := p.uap.p.Parse(req.Device.UA)
		bwFamily = parsedUA.UserAgent.Family
		bwVersion = formatVersion(parsedUA.UserAgent.Major, parsedUA.UserAgent.Minor, parsedUA.UserAgent.Patch)
		osFamily = parsedUA.Os.Family
		osVersion = formatVersion(parsedUA.Os.Major, parsedUA.Os.Minor, parsedUA.Os.Patch)
	}

	// parse Device
	parsedDevice := p.parseDevice(req.Device)

	// parse country from user geo object
	parsedCountry := p.parseCountry(req.User.Geo)

	return &AuctionMetadata{
		device:         parsedDevice,
		country:        parsedCountry,
		domain:         req.Site.Domain,
		href:           req.Site.Page,
		auctionID:      req.ID,
		browserVersion: bwFamily,
		browserName:    bwVersion,
		osVersion:      osVersion,
		osName:         osFamily,
		refresh:        false, // refresh is not available here
	}
}

func (p *Parser) parseDevice(dev *openrtb.Device) string {
	device := "other"

	if dev == nil {
		return device
	}

	switch dev.DeviceType {
	case openrtb.DeviceTypeMobileTablet:
		device = "mobile"
	case openrtb.DeviceTypePersonalComputer:
		device = "desktop"
	case openrtb.DeviceTypeConnectedTV:
		device = "other"
	case openrtb.DeviceTypePhone:
		device = "mobile"
	case openrtb.DeviceTypeTablet:
		device = "tablet"
	case openrtb.DeviceTypeConnectedDevice:
		device = "other"
	case openrtb.DeviceTypeSetTopBox:
		device = "other"
	}
	return device
}

func (p *Parser) parseCountry(geo *openrtb.Geo) string {
	if geo == nil {
		return "unknown"
	}
	if iso2, ok := countries[geo.Country]; ok {
		return iso2
	}
	return "unknown"
}

func (p *Parser) buildAuction(meta *AuctionMetadata, impId string, reqSizes []string) model.Auction {
	auction := model.Auction{
		ScopeId:        p.scope,
		TagId:          "placeholder",
		AdUnit:         "placeholder",
		Device:         meta.device,
		Country:        meta.country,
		Domain:         meta.domain,
		AuctionId:      strings.Join([]string{meta.auctionID, impId}, ""),
		BidRequests:    nil,
		Refresh:        false,
		Tags:           []string{},
		BrowserName:    meta.browserName,
		BrowserVersion: meta.browserVersion,
		OsName:         meta.osName,
		OsVersion:      meta.osVersion,
		Href:           meta.href,
		Sizes:          reqSizes,
	}
	return auction
}

func formatVersion(major, minor, patch string) string {
	res := make([]string, 0)
	if len(major) > 0 {
		res = append(res, major)
	}
	if len(minor) > 0 {
		res = append(res, major)
	}
	if len(patch) > 0 {
		res = append(res, major)
	}
	return strings.Join(res, ".")
}
