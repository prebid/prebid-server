package helpers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]MileAnalyticsEvent, error) {
	//var logEntry *MileAnalyticsEvent
	defer sentry.Recover()

	var events []MileAnalyticsEvent
	if ao != nil {
		if ao.RequestWrapper != nil {
			for _, imp := range ao.RequestWrapper.Imp {

				var bidBiders []string
				var winningBidder, winningSize string
				var winningPrice float64 = 0
				biddersFloorMap := make(map[string]string)

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

									// Extract floor value from bid response
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
					i++
				}
				var respExt RespExt
				err = json.Unmarshal(ao.Response.Ext, &respExt)
				if err != nil {
					return nil, err
				}

				if ao.RequestWrapper != nil {

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
						FloorPrice:        imp.BidFloor,
						MetaData: map[string][]string{
							"prebid_server": []string{"1"},
							"amp":           []string{"1"},
						},
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

									// Extract floor value from bid response
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
					i++
				}

				var noBidBidders []string
				for bidder, _ := range impExt.Prebid.Bidder {
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
						MetaData: map[string][]string{
							"prebid_server": []string{"1"},
							"amp":           []string{"1"},
						},
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
