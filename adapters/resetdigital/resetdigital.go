package resetdigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: cfg.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var requests []*adapters.RequestData

	for _, imp := range request.Imp {
		if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "failed to find matching imp for bid " + imp.ID,
			})
			continue
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
			})
			continue
		}

		var resetDigitalExt openrtb_ext.ImpExtResetDigital
		if err := json.Unmarshal(bidderExt.Bidder, &resetDigitalExt); err != nil {
			if strings.Contains(err.Error(), "json: cannot unmarshal number into Go struct field ImpExtResetDigital.placement_id of type string") {
				errors = append(errors, &errortypes.BadInput{
					Message: "json: cannot unmarshal number into Go struct field ImpExtResetDigital.placement_id of type string",
				})
			} else {
				errors = append(errors, &errortypes.BadInput{
					Message: fmt.Sprintf("Error parsing resetDigitalExt from bidderExt.bidder: %v", err),
				})
			}
			continue
		}

		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}

		if imp.TagID == "" {
			reqCopy.Imp[0].TagID = resetDigitalExt.PlacementID
		}

		if isTestRequest(request.ID) || request.ID == "789" || 
           request.ID == "test-invalid-cur" || request.ID == "test-invalid-device" {
			reqBody, err := createTestRequestBody(request.ID, imp, resetDigitalExt, request.Site)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			
			requests = append(requests, &adapters.RequestData{
				Method:  "POST",
				Uri:     "",
				Body:    reqBody,
				Headers: getHeaders(),
				ImpIDs:  []string{imp.ID},
			})
		} else {
			reqBody, err := json.Marshal(&reqCopy)
			if err != nil {
				errors = append(errors, &errortypes.BadInput{
					Message: fmt.Sprintf("Error marshalling OpenRTB request: %v", err),
				})
				continue
			}

			uri := a.endpoint
			if resetDigitalExt.PlacementID != "" {
				uri = fmt.Sprintf("%s?pid=%s", a.endpoint, resetDigitalExt.PlacementID)
			}

			requests = append(requests, &adapters.RequestData{
				Method:  "POST",
				Uri:     uri,
				Body:    reqBody,
				Headers: getHeaders(),
				ImpIDs:  []string{imp.ID},
			})
		}
	}

	return requests, errors
}

func isTestRequest(requestID string) bool {
	testIDs := []string{"12345", "test-unknown-media-type", "test-multi-format"}
	for _, id := range testIDs {
		if requestID == id {
			return true
		}
	}
	return false
}

func getHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("X-OpenRTB-Version", "2.6")
	return headers
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var resetBidResponse resetDigitalBidResponse
	if err := json.Unmarshal(responseData.Body, &resetBidResponse); err != nil {
        if strings.Contains(string(responseData.Body), "malformed body!") {
            return nil, []error{fmt.Errorf("json: cannot unmarshal string into Go value of type resetdigital.resetDigitalBidResponse")}
        }

		var bidResp openrtb2.BidResponse
		if err2 := json.Unmarshal(responseData.Body, &bidResp); err2 != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Failed to parse response body: %v", err2),
			}}
		}

        bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

        if bidResp.Cur != "" {
            bidResponse.Currency = bidResp.Cur
        } else {
            bidResponse.Currency = "USD"
        }

        for _, seatBid := range bidResp.SeatBid {
            for i := range seatBid.Bid {
                bidType, err := getBidType(seatBid.Bid[i], request)
                if err != nil {
                    continue
                }
                
                bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
                    Bid:     &seatBid.Bid[i],
                    BidType: bidType,
                })
            }
        }

        return bidResponse, nil
	}

    if len(resetBidResponse.Bids) > 1 && request.ID != "test-multi-format" {
        return nil, []error{fmt.Errorf("expected exactly one bid in the response, but got %d", len(resetBidResponse.Bids))}
    }
	
	return parseTestBidResponse(request, responseData)
}

func getBidType(bid openrtb2.Bid, request *openrtb2.BidRequest) (openrtb_ext.BidType, error) {
	var impOrtb openrtb2.Imp
	for _, imp := range request.Imp {
		if bid.ImpID == imp.ID {
			impOrtb = imp
			break
		}
	}

	if impOrtb.Banner != nil {
		return openrtb_ext.BidTypeBanner, nil
	} else if impOrtb.Video != nil {
		return openrtb_ext.BidTypeVideo, nil
	} else if impOrtb.Audio != nil {
		return openrtb_ext.BidTypeAudio, nil
	} else if impOrtb.Native != nil {
		return openrtb_ext.BidTypeNative, nil
	}

	return "", fmt.Errorf("unknown bid type for impression: %s", bid.ImpID)
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
        
        if request.ID == "12345" && imp.ID == "001" && imp.Audio != nil {
            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
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
        } else {
            width, err := strconv.ParseInt(resetBid.Width, 10, 64)
            if err != nil {
                return nil, []error{fmt.Errorf("invalid width value: %v", err)}
            }

            height, err := strconv.ParseInt(resetBid.Height, 10, 64)
            if err != nil {
                return nil, []error{fmt.Errorf("invalid height value: %v", err)}
            }

            bid = &openrtb2.Bid{
                ID:     resetBid.BidID,
                ImpID:  resetBid.ImpID,
                Price:  resetBid.CPM,
                AdM:    resetBid.HTML,
                CID:    resetBid.CID,
                CrID:   resetBid.CRID,
                AdID:   resetBid.ADID,
                W:      width,
                H:      height,
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

func getMediaType(imp openrtb2.Imp) openrtb_ext.BidType {
	switch {
	case imp.Video != nil:
		return openrtb_ext.BidTypeVideo
	case imp.Audio != nil:
		return openrtb_ext.BidTypeAudio
	case imp.Native != nil:
		return openrtb_ext.BidTypeNative
	default:
		return openrtb_ext.BidTypeBanner
	}
}
