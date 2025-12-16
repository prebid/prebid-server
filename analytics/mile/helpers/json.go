package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// impressionBidderData holds data extracted for a single impression
type impressionBidderData struct {
	configuredBidders    []string
	biddersFloorMap      map[string]string
	biddersFloorPriceMap map[string]float64
	gpID                 string
}

// bidResponseData holds data extracted from bid responses
type bidResponseData struct {
	bidBiders     []string
	winningBidder string
	winningSize   string
	winningPrice  float64
}

// processImpressionBidders processes configured bidders and initializes floor maps
func processImpressionBidders(imp *openrtb_ext.ImpWrapper, bidderFloors map[string]map[string]float64) (*impressionBidderData, error) {
	var confBidders ImpressionsExt
	if err := json.Unmarshal(imp.Ext, &confBidders); err != nil {
		return nil, err
	}

	configuredBidders := make([]string, 0, len(confBidders.Prebid.Bidder))
	biddersFloorMap := make(map[string]string)
	biddersFloorPriceMap := make(map[string]float64)
	gpID := confBidders.GPID

	for bidderName := range confBidders.Prebid.Bidder {
		configuredBidders = append(configuredBidders, bidderName)

		// Try to get bidder-specific floor from hooks first
		if impFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
			if floorVal, hasBidderFloor := impFloors[bidderName]; hasBidderFloor {
				biddersFloorMap[bidderName] = fmt.Sprintf("%f", floorVal)
			} else if imp.BidFloor > 0 {
				// Fallback to request floor if bidder-specific floor not available from hooks
				biddersFloorMap[bidderName] = fmt.Sprintf("%f", imp.BidFloor)
			}
		} else if imp.BidFloor > 0 {
			// Fallback to request floor if bidder-specific floor not available
			biddersFloorMap[bidderName] = fmt.Sprintf("%f", imp.BidFloor)
		}
	}

	return &impressionBidderData{
		configuredBidders:    configuredBidders,
		biddersFloorMap:      biddersFloorMap,
		biddersFloorPriceMap: biddersFloorPriceMap,
		gpID:                 gpID,
	}, nil
}

// processBidResponse processes bid responses to extract winning bidder and floor prices
func processBidResponse(impID string, response *openrtb2.BidResponse, bidderData *impressionBidderData) *bidResponseData {
	result := &bidResponseData{
		bidBiders:     []string{},
		winningBidder: "",
		winningSize:   "",
		winningPrice:  0,
	}

	if response == nil || response.SeatBid == nil {
		return result
	}

	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			if bid.ImpID != impID {
				continue
			}

			result.bidBiders = append(result.bidBiders, seatBid.Seat)
			result.winningBidder = seatBid.Seat

			if bid.Price > result.winningPrice {
				result.winningPrice = bid.Price
				result.winningBidder = seatBid.Seat
			}

			// Extract floor value from bid response if available
			if bid.Ext != nil {
				var extBid openrtb_ext.ExtBid
				if err := json.Unmarshal(bid.Ext, &extBid); err == nil {
					if extBid.Prebid != nil && extBid.Prebid.Floors != nil {
						floorValue := extBid.Prebid.Floors.FloorValue
						bidderData.biddersFloorMap[seatBid.Seat] = fmt.Sprintf("%f", floorValue)
						bidderData.biddersFloorPriceMap[seatBid.Seat] = floorValue
					}
				}
			}
		}
	}

	return result
}

// buildMetadata builds metadata from floor information
func buildFloorMeta(bidderFloors map[string]map[string]float64, impID string, requestMeta map[string]string) map[string]map[string]string {
	floorMeta := map[string]map[string]string{
		"requestMeta": requestMeta,
	}

	// Add floor values for each SSP per impression from bidderFloors
	if sspFloors, hasImpFloors := bidderFloors[impID]; hasImpFloors {
		// Initialize sspFloors map if it doesn't exist
		if _, exists := floorMeta["sspFloors"]; !exists {
			floorMeta["sspFloors"] = make(map[string]string)
		}
		for sspName, floorVal := range sspFloors {
			floorMeta["sspFloors"][sspName] = fmt.Sprintf("%f", floorVal)
		}
	}

	return floorMeta
}

// calculateActualFloorPrice calculates the actual floor price from response or hooks
func calculateActualFloorPrice(
	imp *openrtb_ext.ImpWrapper,
	winningBidder string,
	bidderFloors map[string]map[string]float64,
	biddersFloorPriceMap map[string]float64,
) float64 {
	actualFloorPrice := imp.BidFloor

	if winningBidder == "" {
		return actualFloorPrice
	}

	// First try to get floor from response (most accurate - actual floor applied)
	if floorPrice, hasFloor := biddersFloorPriceMap[winningBidder]; hasFloor && floorPrice > 0 {
		return floorPrice
	}

	// Fallback to floor from hooks
	if impFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
		if floorVal, hasBidderFloor := impFloors[winningBidder]; hasBidderFloor {
			return floorVal
		}
	}

	return actualFloorPrice
}

// buildMileAnalyticsEvent builds the final MileAnalyticsEvent from all components
func buildMileAnalyticsEvent(
	requestWrapper *openrtb_ext.RequestWrapper,
	imp *openrtb_ext.ImpWrapper,
	bidderData *impressionBidderData,
	bidResponse *bidResponseData,
	actualFloorPrice float64,
	respExt RespExt,
	floorMetadata map[string]map[string]string,
) MileAnalyticsEvent {
	return MileAnalyticsEvent{
		Ip:              requestWrapper.Device.IP,
		Timestamp:       time.Now().UTC().Unix() * 1000,
		ServerTimestamp: -1,
		Ua:              requestWrapper.Device.UA,
		ArbitraryData:   "",
		Device:          requestWrapper.Device.Model,
		// Publisher:         requestWrapper.Site.Publisher.Name,
		Site:              requestWrapper.Site.Domain,
		ReferrerURL:       requestWrapper.Site.Ref,
		AdvertiserName:    "",
		AuctionID:         requestWrapper.ID,
		Page:              requestWrapper.Site.Page,
		YetiSiteID:        requestWrapper.Site.ID,
		YetiPublisherID:   requestWrapper.Site.Publisher.ID,
		SessionID:         "",
		EventType:         "pbs_agg_adunit",
		Section:           "",
		BidBidders:        bidResponse.bidBiders,
		Cpm:               bidResponse.winningPrice,
		ConfiguredBidders: bidderData.configuredBidders,
		TimedOutBidder:    respExt.getTimeoutBidders(requestWrapper.TMax),
		WinningBidder:     bidResponse.winningBidder,
		WinningSize:       bidResponse.winningSize,
		HasBid:            len(bidResponse.bidBiders) > 0,
		FloorPrice:        actualFloorPrice,
		ConfiguredTimeout: requestWrapper.TMax,
		SiteUID:           floorMetadata["prebid_server"]["site_uid"],
		BiddersFloorMeta:  floorMetadata,
		IsPBS:             true,
	}
}

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]MileAnalyticsEvent, error) {
	defer sentry.Recover()

	if ao == nil || ao.RequestWrapper == nil {
		return []MileAnalyticsEvent{}, nil
	}

	// Extract floor metadata from hook execution outcomes
	floorMetadata := extractFloorMetadataFromHooks(ao.HookExecutionOutcome)

	// Extract bidder-specific floors from hook execution outcomes
	bidderFloors := extractBidderFloorsFromHooks(ao.HookExecutionOutcome)

	// Combine floorMetadata and bidderFloors into a single map for easier downstream use
	if floorMetadata == nil {
		floorMetadata = make(map[string]string)
	}

	// Parse response extension
	var respExt RespExt
	if ao.Response != nil && ao.Response.Ext != nil {
		if err := json.Unmarshal(ao.Response.Ext, &respExt); err != nil {
			return nil, err
		}
	}

	var events []MileAnalyticsEvent
	for _, imp := range ao.RequestWrapper.GetImp() {
		// Process configured bidders and initialize floor maps
		bidderData, err := processImpressionBidders(imp, bidderFloors)
		if err != nil {
			return nil, err
		}

		// Process bid responses to extract winning bidder and floor prices
		bidResponse := processBidResponse(imp.ID, ao.Response, bidderData)

		// Build metadata from floor information
		floorMetadata := buildFloorMeta(bidderFloors, imp.ID, floorMetadata)
		floorMetadata["requestMeta"]["gpID"] = bidderData.gpID
		fmt.Println("floorMetadata is", floorMetadata)

		// Calculate actual floor price from response or hooks
		actualFloorPrice := calculateActualFloorPrice(
			imp,
			bidResponse.winningBidder,
			bidderFloors,
			bidderData.biddersFloorPriceMap,
		)

		// Build the final analytics event
		event := buildMileAnalyticsEvent(
			ao.RequestWrapper,
			imp,
			bidderData,
			bidResponse,
			actualFloorPrice,
			respExt,
			floorMetadata,
		)

		events = append(events, event)
	}

	return events, nil
}

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if vo != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) ([]MileAnalyticsEvent, error) {
	defer sentry.Recover()

	var events []MileAnalyticsEvent
	if ao != nil {
		// Extract floor metadata from hook execution outcomes
		floorMetadata := extractFloorMetadataFromHooks(ao.HookExecutionOutcome)

		// Extract bidder-specific floors from hook execution outcomes
		bidderFloors := extractBidderFloorsFromHooks(ao.HookExecutionOutcome)

		if ao.RequestWrapper != nil {
			for _, imp := range ao.RequestWrapper.Imp {

				var sizeRequested []string
				for _, format := range imp.Banner.Format {
					size := fmt.Sprintf("%dx%d", format.W, format.H)
					sizeRequested = append(sizeRequested, size)
				}

				var bidBiders []string
				sizePrize := make(map[string]map[string]float64)
				bidbiddersMap := make(map[string]struct{})
				var winningBidder, winningSize string
				var winningPrice float64 = 0.0
				biddersFloorMap := make(map[string]string)
				biddersFloorPriceMap := make(map[string]float64) // Store actual floor prices from response as float64

				// First, get configured bidders and initialize floor map with request floor
				var impExt ImpressionsExt
				//if ao.RequestWrapper != nil {
				err := json.Unmarshal(imp.Ext, &impExt)
				if err != nil {
					return nil, err
					//}
				}
				configuredBidders := make([]string, len(impExt.Prebid.Bidder))
				i := 0
				for k := range impExt.Prebid.Bidder {
					configuredBidders[i] = k

					// Try to get bidder-specific floor from hooks first
					if impFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
						if floorVal, hasBidderFloor := impFloors[k]; hasBidderFloor {
							biddersFloorMap[k] = fmt.Sprintf("%f", floorVal)
						} else if imp.BidFloor > 0 {
							// Fallback to request floor if bidder-specific floor not available from hooks
							biddersFloorMap[k] = fmt.Sprintf("%f", imp.BidFloor)
						}
					} else if imp.BidFloor > 0 {
						// Fallback to request floor if bidder-specific floor not available
						biddersFloorMap[k] = fmt.Sprintf("%f", imp.BidFloor)
					}
					i++
				}

				// Evaluate bids
				if ao.AuctionResponse != nil {
					if ao.AuctionResponse.SeatBid != nil {

						for _, seatBid := range ao.AuctionResponse.SeatBid {
							for _, bid := range seatBid.Bid {
								if bid.ImpID == imp.ID {

									size := fmt.Sprintf("%dx%d", bid.W, bid.H)
									sizePrize[seatBid.Seat] = map[string]float64{size: bid.Price}

									bidBiders = append(bidBiders, seatBid.Seat)
									bidbiddersMap[seatBid.Seat] = struct{}{}
									winningBidder = seatBid.Seat
									if bid.Price > winningPrice {
										winningPrice = bid.Price
										winningBidder = seatBid.Seat
										//winningSize = bid.Ext.
									}

									// Override with floor value from bid response if available
									if bid.Ext != nil {
										var extBid openrtb_ext.ExtBid
										if err := json.Unmarshal(bid.Ext, &extBid); err == nil {
											if extBid.Prebid != nil && extBid.Prebid.Floors != nil {
												floorValue := extBid.Prebid.Floors.FloorValue
												biddersFloorMap[seatBid.Seat] = fmt.Sprintf("%f", floorValue)
												biddersFloorPriceMap[seatBid.Seat] = floorValue // Store as float64 for actual floor price
											}
										}
									}
								}
							}
						}
					}
				}

				var noBidBidders []string
				for bidder := range impExt.Prebid.Bidder {
					if _, found := bidbiddersMap[bidder]; !found {
						noBidBidders = append(noBidBidders, bidder)
					}

				}

				var respExt RespExt
				err = json.Unmarshal(ao.AuctionResponse.Ext, &respExt)
				if err != nil {
					return nil, err
				}
				if respExt.Errors != nil {
					for bidder, err := range respExt.Errors {
						errString := fmt.Errorf("%v: %+v", bidder, err)
						sentry.CaptureException(errString)
					}
				}

				if ao.RequestWrapper != nil {

					// Build metadata from floor metadata extracted from hooks
					metadata := map[string][]string{
						"prebid_server": {"1"},
						"amp":           {"1"},
					}
					// Add floor metadata (site_uid, country, platform, floor_url)
					for k, v := range floorMetadata {
						metadata[k] = []string{v}
					}

					// Add floor values for each SSP per impression from bidderFloors
					if sspFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
						for sspName, floorVal := range sspFloors {
							metadataKey := fmt.Sprintf("floor_%s", sspName)
							metadata[metadataKey] = []string{fmt.Sprintf("%f", floorVal)}
						}
					}

					// Get actual floor price from ORTB response
					actualFloorPrice := imp.BidFloor
					if winningBidder != "" {
						// First try to get floor from response (most accurate - actual floor applied)
						if floorPrice, hasFloor := biddersFloorPriceMap[winningBidder]; hasFloor && floorPrice > 0 {
							actualFloorPrice = floorPrice
						} else if impFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
							// Fallback to floor from hooks
							if floorVal, hasBidderFloor := impFloors[winningBidder]; hasBidderFloor {
								actualFloorPrice = floorVal
							}
						}
					}

					logEntry := MileAnalyticsEvent{
						//SessionID: ao.RequestWrapper
						Ip:              ao.RequestWrapper.Device.IP,
						Timestamp:       time.Now().UTC().Unix() * 1000,
						ServerTimestamp: -1,
						GptAdUnit:       imp.ID,
						//ClientVersion: ao.RequestWrapper.Ext.
						Ua:            ao.RequestWrapper.Device.UA,
						ArbitraryData: "",
						Device:        ao.RequestWrapper.Device.Model,
						// Publisher:         ao.RequestWrapper.Site.Publisher.Domain,
						Site:              ao.RequestWrapper.Site.Domain,
						ReferrerURL:       ao.RequestWrapper.Site.Ref,
						AdvertiserName:    "",
						AuctionID:         ao.RequestWrapper.ID,
						Page:              ao.RequestWrapper.Site.Page,
						YetiSiteID:        ao.RequestWrapper.Site.ID,
						YetiPublisherID:   ao.RequestWrapper.Site.Publisher.ID,
						SessionID:         "",
						EventType:         "pbs_agg_adunit",
						Section:           "",
						SizesRequested:    sizeRequested,
						BidBidders:        bidBiders,
						ConfiguredBidders: configuredBidders,
						NoBidBidders:      noBidBidders,
						TimedOutBidder:    respExt.getTimeoutBidders(ao.RequestWrapper.TMax),
						WinningBidder:     winningBidder,
						SizePrice:         sizePrize,
						Cpm:               winningPrice,
						WinningSize:       winningSize,
						ConfiguredTimeout: ao.RequestWrapper.TMax,
						ResponseTimes:     respExt.ResponseTimeMillis,
						FloorPrice:        actualFloorPrice,
						MetaData:          metadata,
						BiddersFloorMeta: map[string]map[string]string{
							"prebid_server": biddersFloorMap,
						},
						//Viewability: ao.RequestWrapper.
						//WinningSize: ao.Response.SeatBi
						IsPBS: true,
					}

					events = append(events, logEntry)
				}
			}
		}

	}
	if ao.Errors != nil {
		if len(ao.Errors) > 0 {
			err := fmt.Errorf("%v", ao.Errors)
			return events, err
		}
	}

	//events = append(events, logEntry)
	return events, nil

}
func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if cso != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}
func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) (*MileAnalyticsEvent, error) {
	var logEntry *MileAnalyticsEvent
	if so != nil {
		//var request *openrtb2.BidRequest
		//if ao.RequestWrapper != nil {
		//	request = ao.RequestWrapper.BidRequest
		//}
		logEntry = &MileAnalyticsEvent{
			//Status:               ao.Status,
			//Errors:               ao.Errors,
			//Request:              request,
			//Response:             ao.Response,
			//Account:              ao.Account,
			//StartTime:            ao.StartTime,
			//HookExecutionOutcome: ao.HookExecutionOutcome,
		}
	}
	return logEntry, nil

}

// extractFloorMetadataFromHooks extracts floor-related data from hook execution outcomes
func extractFloorMetadataFromHooks(outcomes []hookexecution.StageOutcome) map[string]string {
	metadata := make(map[string]string)

	for _, stageOutcome := range outcomes {
		for _, group := range stageOutcome.Groups {
			for _, invocation := range group.InvocationResults {
				// Look for floor-injection activity from mile.floors module
				if invocation.HookID.ModuleCode == "mile.floors" {
					for _, activity := range invocation.AnalyticsTags.Activities {
						if activity.Name == "floor-injection" {
							for _, result := range activity.Results {
								if result.Values != nil {
									// Merge values into metadata
									for k, v := range result.Values {
										metadata[k] = v.(string)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return metadata
}

// extractBidderFloorsFromHooks extracts bidder-specific floor values from hook execution outcomes
// Returns map[impression_id] -> map[bidder] -> floor_value
func extractBidderFloorsFromHooks(outcomes []hookexecution.StageOutcome) map[string]map[string]float64 {
	// Map of impression_id -> bidder -> floor value
	impressionBidderFloors := make(map[string]map[string]float64)

	for _, stageOutcome := range outcomes {
		for _, group := range stageOutcome.Groups {
			for _, invocation := range group.InvocationResults {
				// Look for bidder-floors activity from mile.floors module
				if invocation.HookID.ModuleCode == "mile.floors" {
					for _, activity := range invocation.AnalyticsTags.Activities {
						if activity.Name == "bidder-floors" {
							for _, result := range activity.Results {
								if result.Values != nil && result.AppliedTo.Bidder != "" {
									// result.Values is map[impression_id] -> map[bidder] -> {floor, currency}
									for impID, bidderData := range result.Values {
										// impID is already a string key from the map
										// Initialize impression map if needed
										if _, exists := impressionBidderFloors[impID]; !exists {
											impressionBidderFloors[impID] = make(map[string]float64)
										}

										// Extract bidder floor data
										if bidderMap, ok := bidderData.(map[string]interface{}); ok {
											for bidderName, floorData := range bidderMap {
												if floorMap, ok := floorData.(map[string]interface{}); ok {
													if floorVal, ok := floorMap["floor"].(float64); ok {
														impressionBidderFloors[impID][bidderName] = floorVal
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return impressionBidderFloors
}
