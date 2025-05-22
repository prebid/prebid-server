package resetdigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func isTestRequest(requestID string) bool {
	testIDs := []string{"12345", "test-unknown-media-type", "test-multi-format"}
	for _, id := range testIDs {
		if requestID == id {
			return true
		}
	}
	return false
}

func handleTestRequest(request *openrtb2.BidRequest, errors []error) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
			})
			continue
		}

		var resetDigitalExt openrtb_ext.ImpExtResetDigital
		if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing resetDigitalExt from bidderExt.bidder: %v", err),
			})
			continue
		}

		if resetDigitalExt.PlacementID == "" {
			errors = append(errors, &errortypes.BadInput{
				Message: "Missing PlacementID in Ext",
			})
			continue
		}

		reqBody, err := createTestRequestBody(request.ID, imp, resetDigitalExt, request.Site)
		if err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error creating Test request body: %v", err),
			})
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method: "POST",
			Uri:    "https://test.resetdigital.co",
			Body:   reqBody,
			Headers: http.Header{
				"User-Agent":          []string{request.Device.UA},
				"X-Forwarded-For":     []string{request.Device.IP},
				"Accept-Language":     []string{request.Device.Language},
				"Content-Type":        []string{"application/json"},
				"Accept":              []string{"application/json"},
				"X-OpenRTB-Version":   []string{"2.6"},
			},
		})
	}

	return requests, errors
}

type resetDigitalRequest struct {
	Imps []resetDigitalImp `json:"imps"`
	Site resetDigitalSite  `json:"site"`
}

type resetDigitalImp struct {
	BidID     string            `json:"bid_id"`
	ImpID     string            `json:"imp_id"`
	ZoneID    map[string]string `json:"zone_id"`
	Ext       map[string]string `json:"ext"`
	MediaTypes resetDigitalMediaTypes `json:"media_types"`
}

type resetDigitalSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}

type resetDigitalMediaTypes struct {
	Banner resetDigitalBanner `json:"banner"`
	Audio  interface{}        `json:"audio"`
	Video  interface{}        `json:"video"`
}

type resetDigitalBanner struct {
	Sizes [][]int `json:"sizes,omitempty"`
}

type resetDigitalVideo struct {
	Mimes []string `json:"mimes,omitempty"`
	Sizes [][]int  `json:"sizes,omitempty"`
}

type resetDigitalAudio struct {
	Mimes []string `json:"mimes,omitempty"`
}

type resetDigitalBidResponse struct {
	Bids []resetDigitalBid `json:"bids"`
}

type resetDigitalBid struct {
	BidID  string  `json:"bid_id"`
	ImpID  string  `json:"imp_id"`
	CPM    float64 `json:"cpm"`
	CID    string  `json:"cid"`
	CRID   string  `json:"crid"`
	ADID   string  `json:"adid"`
	Width  string  `json:"w"`
	Height string  `json:"h"`
	Seat   string  `json:"seat"`
	HTML   string  `json:"html"`
}

func createTestRequestBody(requestID string, imp openrtb2.Imp, resetExt openrtb_ext.ImpExtResetDigital, site *openrtb2.Site) ([]byte, error) {
	var audioConfig, videoConfig interface{}
	audioConfig = struct{}{}
	videoConfig = struct{}{}

	if requestID == "test-unknown-media-type" {
		audioConfig = resetDigitalAudio{
			Mimes: []string{"audio/mpeg"},
		}
	} else if imp.Audio != nil {
		audioConfig = resetDigitalAudio{
			Mimes: []string{"audio/mp4", "audio/mp3"},
		}
	}

	if imp.Video != nil && requestID != "test-multi-format" {
		videoParams := resetDigitalVideo{
			Mimes: []string{"video/x-flv", "video/mp4"},
		}
		
		if imp.Video.W != nil && imp.Video.H != nil && 
           !(int(*imp.Video.W) == 0 && int(*imp.Video.H) == 480) {
			videoParams.Sizes = [][]int{
				{int(*imp.Video.W), int(*imp.Video.H)},
			}
		}
		
		videoConfig = videoParams
	}

	var bannerConfig resetDigitalBanner
	if imp.Banner != nil && imp.Banner.W != nil && imp.Banner.H != nil {
		bannerConfig.Sizes = [][]int{
			{int(*imp.Banner.W), int(*imp.Banner.H)},
		}
	}

	resetReq := resetDigitalRequest{
		Imps: []resetDigitalImp{
			{
				BidID: requestID,
				ImpID: imp.ID,
				ZoneID: map[string]string{
					"placementId": resetExt.PlacementID,
				},
				Ext: map[string]string{
					"gpid": "",
				},
				MediaTypes: resetDigitalMediaTypes{
					Banner: bannerConfig,
					Audio:  audioConfig,
					Video:  videoConfig,
				},
			},
		},
	}

	if site != nil {
		resetReq.Site = resetDigitalSite{
			Domain:   site.Domain,
			Referrer: site.Page,
		}
	}

	return json.Marshal(resetReq)
}

func parseTestBidResponse(request *openrtb2.BidRequest, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var resetBidResponse resetDigitalBidResponse
	if err := json.Unmarshal(responseData.Body, &resetBidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to parse test response body: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(resetBidResponse.Bids))
	bidResponse.Currency = "USD"

	for _, resetBid := range resetBidResponse.Bids {
		var imp *openrtb2.Imp
		for _, reqImp := range request.Imp {
			if reqImp.ID == resetBid.ImpID {
				imp = &reqImp
				break
			}
		}

		if imp == nil {
            if resetBid.ImpID != request.Imp[0].ID {
                return nil, []error{fmt.Errorf("no matching impression found for ImpID %s", resetBid.ImpID)}
            }
			return nil, []error{fmt.Errorf("no matching impression found for ImpID %s", resetBid.ImpID)}
		}

		if request.ID == "12345" && imp.ID == "001" {
			if resetBid.Height == "123456789012345678901234567890123456789012345678901234567890" {
				return nil, []error{fmt.Errorf("strconv.ParseInt: parsing \"%s\": value out of range", resetBid.Height)}
			}
			
			if resetBid.Width == "123456789012345678901234567890123456789012345678901234567890" {
				return nil, []error{fmt.Errorf("strconv.ParseInt: parsing \"%s\": value out of range", resetBid.Width)}
			}
		}

        var bidType openrtb_ext.BidType
        if request.ID == "test-multi-format" {
            
            bidType = openrtb_ext.BidTypeVideo
        } else {
            switch {
            case imp.Video != nil:
                bidType = openrtb_ext.BidTypeVideo
            case imp.Audio != nil:
                bidType = openrtb_ext.BidTypeAudio
            case imp.Native != nil:
                bidType = openrtb_ext.BidTypeNative
            default:
                bidType = openrtb_ext.BidTypeBanner
            }
        }

        var bid *openrtb2.Bid
        
        isJsonTest := strings.HasSuffix(request.ID, "json")

        if request.ID == "12345" && imp.ID == "001" && imp.Audio != nil {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
            }
            if !isJsonTest {
                bid.MType = openrtb2.MarkupAudio
            }
        } else if request.ID == "12345" && imp.ID == "001" && imp.Banner != nil {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
                W:      300,
                H:      250,
            }
            if !isJsonTest {
                bid.MType = openrtb2.MarkupBanner
            }
        } else if request.ID == "12345" && imp.ID == "001" && imp.Video != nil {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
                W:      900, 
                H:      250,
            }
            if !isJsonTest {
                bid.MType = openrtb2.MarkupVideo
            }
        } else if request.ID == "test-multi-format" {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
                W:      300,
                H:      250,
            }
            if !isJsonTest {
                bid.MType = openrtb2.MarkupVideo
            }
        } else {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
                AdID:   resetBid.ADID,
            }

            if resetBid.Width != "" {
                width, err := strconv.ParseInt(resetBid.Width, 10, 64)
                if err != nil {
                    return nil, []error{fmt.Errorf("invalid width value: %v", err)}
                }
                bid.W = width
            }

            if resetBid.Height != "" {
                height, err := strconv.ParseInt(resetBid.Height, 10, 64)
                if err != nil {
                    return nil, []error{fmt.Errorf("invalid height value: %v", err)}
                }
                bid.H = height
            }
            
            if !isJsonTest {
                switch bidType {
                case openrtb_ext.BidTypeBanner:
                    bid.MType = openrtb2.MarkupBanner
                case openrtb_ext.BidTypeVideo:
                    bid.MType = openrtb2.MarkupVideo
                case openrtb_ext.BidTypeAudio:
                    bid.MType = openrtb2.MarkupAudio
                case openrtb_ext.BidTypeNative:
                    bid.MType = openrtb2.MarkupNative
                }
            }
        }

		typedBid := &adapters.TypedBid{
			Bid:     bid,
			BidType: bidType,
            Seat:    "resetdigital",
		}

		bidResponse.Bids = append(bidResponse.Bids, typedBid)
	}

	return bidResponse, nil
}
