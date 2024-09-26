package adnuntius

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func generateAdResponse(ad Ad, imp openrtb2.Imp, html string, request *openrtb2.BidRequest) (*openrtb2.Bid, []error) {

	creativeWidth, widthErr := strconv.ParseInt(ad.CreativeWidth, 10, 64)
	if widthErr != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Value of width: %s is not a string", ad.CreativeWidth),
		}}
	}

	creativeHeight, heightErr := strconv.ParseInt(ad.CreativeHeight, 10, 64)
	if heightErr != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Value of height: %s is not a string", ad.CreativeHeight),
		}}
	}

	price := ad.Bid.Amount

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
		}}
	}

	var adnuntiusExt openrtb_ext.ImpExtAdnunitus
	if err := json.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
		}}
	}

	if adnuntiusExt.BidType != "" {
		if strings.EqualFold(string(adnuntiusExt.BidType), "net") {
			price = ad.NetBid.Amount
		}
		if strings.EqualFold(string(adnuntiusExt.BidType), "gross") {
			price = ad.GrossBid.Amount
		}
	}

	extJson, err := generateReturnExt(ad, request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error extracting Ext: %s", err.Error()),
		}}
	}

	adDomain := []string{}
	for _, url := range ad.DestinationUrls {
		domainArray := strings.Split(url, "/")
		domain := strings.Replace(domainArray[2], "www.", "", -1)
		adDomain = append(adDomain, domain)
	}

	bid := openrtb2.Bid{
		ID:      ad.AdId,
		ImpID:   imp.ID,
		W:       creativeWidth,
		H:       creativeHeight,
		AdID:    ad.AdId,
		DealID:  ad.DealID,
		CID:     ad.LineItemId,
		CrID:    ad.CreativeId,
		Price:   price * 1000,
		AdM:     html,
		ADomain: adDomain,
		Ext:     extJson,
	}
	return &bid, nil
}

func generateReturnExt(ad Ad, request *openrtb2.BidRequest) (json.RawMessage, error) {
	// We always force the publisher to render
	var adRender int8 = 0

	var requestRegsExt *openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := json.Unmarshal(request.Regs.Ext, &requestRegsExt); err != nil {

			return nil, fmt.Errorf("Failed to parse Ext information in Adnuntius: %v", err)
		}
	}

	if ad.Advertiser.Name != "" && requestRegsExt != nil && requestRegsExt.DSA != nil {
		legalName := ad.Advertiser.Name
		if ad.Advertiser.LegalName != "" {
			legalName = ad.Advertiser.LegalName
		}
		ext := &openrtb_ext.ExtBid{
			DSA: &openrtb_ext.ExtBidDSA{
				AdRender: &adRender,
				Paid:     legalName,
				Behalf:   legalName,
			},
		}
		returnExt, err := json.Marshal(ext)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse Ext information in Adnuntius: %v", err)
		}

		return returnExt, nil
	}
	return nil, nil
}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	var currency string
	adunitMap := map[string]AdUnit{}

	for _, adnRespAdunit := range adnResponse.AdUnits {
		adunitMap[adnRespAdunit.TargetId] = adnRespAdunit
	}

	for _, imp := range request.Imp {

		auId, _, _, err := jsonparser.Get(imp.Ext, "bidder", "auId")
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error at Bidder auId: %s", err.Error()),
			}}
		}

		targetID := fmt.Sprintf("%s-%s", string(auId), imp.ID)
		adunit := adunitMap[targetID]

		if len(adunit.Ads) > 0 {

			ad := adunit.Ads[0]
			currency = ad.Bid.Currency

			adBid, err := generateAdResponse(ad, imp, adunit.Html, request)
			if err != nil {
				return nil, []error{&errortypes.BadInput{
					Message: "Error at ad generation",
				}}
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     adBid,
				BidType: "banner",
			})

			for _, deal := range adunit.Deals {
				dealBid, err := generateAdResponse(deal, imp, deal.Html, request)
				if err != nil {
					return nil, []error{&errortypes.BadInput{
						Message: "Error at ad generation",
					}}
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     dealBid,
					BidType: "banner",
				})
			}

		}

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}
