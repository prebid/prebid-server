package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type RequestExt struct {
	Bidder adnRequestAdunit `json:"bidder"`
}

func setHeaders(ortbRequest openrtb2.BidRequest) http.Header {

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if ortbRequest.Device != nil {
		if ortbRequest.Device.IP != "" {
			headers.Add("X-Forwarded-For", ortbRequest.Device.IP)
		}
		if ortbRequest.Device.UA != "" {
			headers.Add("user-agent", ortbRequest.Device.UA)
		}
	}
	return headers
}

func makeEndpointUrl(ortbRequest openrtb2.BidRequest, a *adapter, noCookies bool) (string, []error) {
	uri, err := url.Parse(a.endpoint)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	gdpr, consent, err := getGDPR(&ortbRequest)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse GDPR information: %v", err)}
	}

	if gdpr != "" {
		extraInfoURI, err := url.Parse(a.extraInfo)
		if err != nil {
			return "", []error{fmt.Errorf("invalid extraInfo URL: %v", err)}
		}
		uri = extraInfoURI
	}

	if !noCookies {
		var deviceExt extDeviceAdnuntius
		if ortbRequest.Device != nil && ortbRequest.Device.Ext != nil {
			if err := jsonutil.Unmarshal(ortbRequest.Device.Ext, &deviceExt); err != nil {
				return "", []error{fmt.Errorf("failed to parse device ext: %v", err)}
			}
		}

		if deviceExt.NoCookies {
			noCookies = true
		}
	}

	_, offset := a.time.Now().Zone()
	tzo := -offset / minutesInHour

	q := uri.Query()
	if gdpr != "" {
		q.Set("gdpr", gdpr)
	}

	if consent != "" {
		q.Set("consentString", consent)
	}

	if noCookies {
		q.Set("noCookies", "true")
	}

	q.Set("tzo", strconv.Itoa(tzo))
	q.Set("format", "prebidServer")

	// Set the query params to the URI
	uri.RawQuery = q.Encode()

	// Return the correctly formatted URL
	return uri.String(), nil
}

func getImpSizes(imp openrtb2.Imp, bidType string) [][]int64 {
	if bidType == "banner" {
		if imp.Banner != nil {
			if len(imp.Banner.Format) > 0 {
				sizes := make([][]int64, len(imp.Banner.Format))
				for i, format := range imp.Banner.Format {
					sizes[i] = []int64{format.W, format.H}
				}

				return sizes
			} else if imp.Banner.W != nil && imp.Banner.H != nil {
				size := make([][]int64, 1)
				size[0] = []int64{*imp.Banner.W, *imp.Banner.H}
				return size
			}
		}
	}

	return nil
}

func getSiteExtAsKv(request *openrtb2.BidRequest) (siteExt, []error) {
	var extSite siteExt
	if request.Site != nil && request.Site.Ext != nil {
		if err := jsonutil.Unmarshal(request.Site.Ext, &extSite); err != nil {
			return extSite, []error{fmt.Errorf("failed to parse site ext in Adnuntius: %v", err)}
		}
	}
	return extSite, nil
}

func getGDPR(request *openrtb2.BidRequest) (string, string, error) {

	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtRegs in Adnuntius GDPR check: %v", err)
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdpr = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtUser in Adnuntius GDPR check: %v", err)
		}
		consent = extUser.Consent
	}

	return gdpr, consent, nil
}

func generateReturnExt(ad Ad, request *openrtb2.BidRequest) (json.RawMessage, error) {
	// We always force the publisher to render
	var adRender int8 = 0

	var requestRegsExt *openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &requestRegsExt); err != nil {

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

		returnExt, err := jsonutil.Marshal(ext)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse Ext information in Adnuntius: %v", err)
		}

		return returnExt, nil
	}
	return nil, nil
}

func generateAdUnit(imp openrtb2.Imp, adnuntiusExt openrtb_ext.ImpExtAdnunitus, bidType string) adnRequestAdunit {
	adUnit := adnRequestAdunit{
		AuId:       adnuntiusExt.Auid,
		TargetId:   fmt.Sprintf("%s-%s:%s", adnuntiusExt.Auid, imp.ID, bidType),
		Dimensions: getImpSizes(imp, bidType),
	}

	if adnuntiusExt.MaxDeals > 0 {
		adUnit.MaxDeals = adnuntiusExt.MaxDeals
	}
	return adUnit
}

func convertMarkupTypeToBidType(markupType openrtb2.MarkupType) openrtb_ext.BidType {
	switch markupType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative
	}
	return openrtb_ext.BidTypeBanner
}
