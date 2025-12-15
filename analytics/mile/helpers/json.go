package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]MileAnalyticsEvent, error) {
	//var logEntry *MileAnalyticsEvent
	defer sentry.Recover()

	var events []MileAnalyticsEvent
	if ao != nil {
		// Extract floor metadata from hook execution outcomes
		floorMetadata := extractFloorMetadataFromHooks(ao.HookExecutionOutcome)
		fmt.Println("floorMetadata is", floorMetadata)

		// Extract bidder-specific floors from hook execution outcomes
		bidderFloors := extractBidderFloorsFromHooks(ao.HookExecutionOutcome)
		fmt.Println("bidderFloors is", bidderFloors)

		if ao.RequestWrapper != nil {
			for _, imp := range ao.RequestWrapper.Imp {

				var bidBiders []string
				var winningBidder, winningSize string
				var winningPrice float64 = 0
				biddersFloorMap := make(map[string]string)

				// First, get configured bidders and initialize floor map with request floor
				var confBidders ImpressionsExt
				//if ao.RequestWrapper != nil {
				err := json.Unmarshal(imp.Ext, &confBidders)
				if err != nil {
					return nil, err
					//}
				}
				configuredBidders := make([]string, len(confBidders.Prebid.Bidder))
				i := 0
				for k := range confBidders.Prebid.Bidder {
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

				if ao.Response != nil {
					if ao.Response.SeatBid != nil {

						for _, seatBid := range ao.Response.SeatBid {
							for _, bid := range seatBid.Bid {
								if bid.ImpID == imp.ID {
									bidBiders = append(bidBiders, seatBid.Seat)
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
												biddersFloorMap[seatBid.Seat] = fmt.Sprintf("%f", extBid.Prebid.Floors.FloorValue)
											}
										}
									}
								}
							}
						}
					}
				}
				var respExt RespExt
				err = json.Unmarshal(ao.Response.Ext, &respExt)
				if err != nil {
					return nil, err
				}

				if ao.RequestWrapper != nil {

					// Build metadata from floor metadata extracted from hooks
					metadata := map[string][]string{
						"prebid_server": []string{"1"},
					}
					// Add floor metadata (site_uid, country, platform, floor_url)
					for k, v := range floorMetadata {
						if strVal, ok := v.(string); ok {
							metadata[k] = []string{strVal}
						}
					}

					// Add floor values for each SSP per impression from bidderFloors
					if sspFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
						for sspName, floorVal := range sspFloors {
							metadataKey := fmt.Sprintf("floor_%s", sspName)
							metadata[metadataKey] = []string{fmt.Sprintf("%f", floorVal)}
						}
					}

					logEntry := MileAnalyticsEvent{
						//SessionID: ao.RequestWrapper
						Ip:              ao.RequestWrapper.Device.IP,
						Timestamp:       time.Now().UTC().Unix() * 1000,
						ServerTimestamp: -1,
						//ClientVersion: ao.RequestWrapper.Ext.
						Ua:                ao.RequestWrapper.Device.UA,
						ArbitraryData:     "",
						Device:            ao.RequestWrapper.Device.Model,
						Publisher:         ao.RequestWrapper.Site.Publisher.Domain,
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
						BidBidders:        bidBiders,
						ConfiguredBidders: configuredBidders,
						TimedOutBidder:    respExt.getTimeoutBidders(ao.RequestWrapper.TMax),
						WinningBidder:     winningBidder,
						WinningSize:       winningSize,
						ConfiguredTimeout: ao.RequestWrapper.TMax,
						// SiteUID:           floorMetadata["site_uid"],
						FloorPrice: imp.BidFloor,
						MetaData:   metadata,
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

	//events = append(events, logEntry)
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
												biddersFloorMap[seatBid.Seat] = fmt.Sprintf("%f", extBid.Prebid.Floors.FloorValue)
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
						"prebid_server": []string{"1"},
						"amp":           []string{"1"},
					}
					// Add floor metadata (site_uid, country, platform, floor_url)
					for k, v := range floorMetadata {
						if strVal, ok := v.(string); ok {
							metadata[k] = []string{strVal}
						}
					}

					// Add floor values for each SSP per impression from bidderFloors
					if sspFloors, hasImpFloors := bidderFloors[imp.ID]; hasImpFloors {
						for sspName, floorVal := range sspFloors {
							metadataKey := fmt.Sprintf("floor_%s", sspName)
							metadata[metadataKey] = []string{fmt.Sprintf("%f", floorVal)}
						}
					}

					logEntry := MileAnalyticsEvent{
						//SessionID: ao.RequestWrapper
						Ip:              ao.RequestWrapper.Device.IP,
						Timestamp:       time.Now().UTC().Unix() * 1000,
						ServerTimestamp: -1,
						GptAdUnit:       imp.ID,
						//ClientVersion: ao.RequestWrapper.Ext.
						Ua:                ao.RequestWrapper.Device.UA,
						ArbitraryData:     "",
						Device:            ao.RequestWrapper.Device.Model,
						Publisher:         ao.RequestWrapper.Site.Publisher.Domain,
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
						FloorPrice:        imp.BidFloor,
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
func extractFloorMetadataFromHooks(outcomes []hookexecution.StageOutcome) map[string]interface{} {
	metadata := make(map[string]interface{})

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
										metadata[k] = v
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
