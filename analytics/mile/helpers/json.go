package helpers

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/prebid/prebid-server/v3/analytics"
	"time"
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
						MetaData: map[string][]string{
							"prebid_server": []string{"1"},
							"amp":           []string{"1"},
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

				var bidBiders []string
				bidbiddersMap := make(map[string]struct{})
				var winningBidder, winningSize string
				var winningPrice float64 = 0.0
				if ao.AuctionResponse != nil {
					if ao.AuctionResponse.SeatBid != nil {

						for _, seatBid := range ao.AuctionResponse.SeatBid {
							for _, bid := range seatBid.Bid {
								if bid.ImpID == imp.ID {
									bidBiders = append(bidBiders, seatBid.Seat)
									bidbiddersMap[seatBid.Seat] = struct{}{}
									winningBidder = seatBid.Seat
									if bid.Price > winningPrice {
										winningPrice = bid.Price
										winningBidder = seatBid.Seat
										//winningSize = bid.Ext.
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
						NoBidBidders:      noBidBidders,
						TimedOutBidder:    respExt.getTimeoutBidders(ao.RequestWrapper.TMax),
						WinningBidder:     winningBidder,
						Cpm:               winningPrice,
						WinningSize:       winningSize,
						ConfiguredTimeout: ao.RequestWrapper.TMax,
						ResponseTimes:     respExt.ResponseTimeMillis,
						MetaData: map[string][]string{
							"prebid_server": []string{"1"},
							"amp":           []string{"1"},
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
		err := fmt.Errorf("%v", ao.Errors)
		return events, err
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
