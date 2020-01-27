package newsiq

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
)

/*
type RequestType string

const (
	COOKIE_SYNC RequestType = "/cookie_sync"
	AUCTION     RequestType = "/openrtb2/auction"
	SETUID      RequestType = "/set_uid"
	AMP         RequestType = "/openrtb2/amp"
)
*/

/*
const STATUS = {
  BID_RECEIVED: 9,
  BID_WON: 10,
  BID_LOST: 11,
  NO_BID: 12,
  BID_TIMEOUT: 13
};

const MSG_TYPE = {
  Auction_Init: 101,
  Bid_Requested: 102,
  Bid_Response: 103,
  Bid_Timeout: 104,
  Bid_Won: 105,
  Bid_Lost: 106
};

let _topLevel = {};
let _queue = [];
let _sent = [];


Q: auction init, requested, response are all instances when we have to pass protobuf data to the data endpoint? Meaning we make 3 different calls for each prebid request?

A: prebid web create an auction instance, inside which it creates multi bidding instance for each bidder. For example, if the the configure says it need 3 bidders: apn, newsiq, rubicon, 3 bid instances will be created
auction init event triggers when the auction instance created
request event triggers when the requests to the 3 bid adapter fires
response event happens when the bid adapter returns bidding data
new messages
we need send protobuf data to the data endpoint to all 3 types of events
but not just 3 calls
auction init -- 1 call
request -- 3 calls (one for each bidder)
response - (0-3) calls
since response could fail, if all success, 3 calls, all fail 0 call
*/

type MsgType int

const (
	AuctionInit  MsgType = 101
	BidRequested MsgType = 102
	BidResponse  MsgType = 103
	BidTimeout   MsgType = 104
	BidWon       MsgType = 105
	BidLost      MsgType = 106
)

/**** PROD Variables ****/

const DebugLogging = true

const PrebidServerVersion = "0.97.0"
const newsIQDataEndpointDev = "https://newscorp-newsiq-dev.appspot.com/pb/"
const newsIQDataEndpointProd = "https://log.ncaudienceexchange.com/pb"

/**** PROD Variables ****/

func (d *DataLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogAuctionObject")
	}
}
func (d *DataLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogSetUIDObject")
	}
}
func (d *DataLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogCookieSyncObject")
	}
}
func (d *DataLogger) LogAmpObject(ao *analytics.AmpObject) {
	if DebugLogging {
		fmt.Println("News IQ Module - LogAmpObject")
	}
}

/* TODO : Build multithreaded env like DataLog: https://www.opsdash.com/blog/job-queues-in-go.html, https://nesv.github.io/golang/2014/02/25/worker-queues-in-go.html
func worker(jobChan <-chan Job) {
    for job := range jobChan {
        SendCollectorData(job)
    }
}

// make a channel with a capacity of 100.
jobChan := make(chan Job, 100)

// start the worker
go worker(jobChan)

func TrySendCollectorData(job Job, jobChan <-chan Job) bool {
    select {
    case jobChan <- job:
        return true
    default:
        return false
    }
}*/

func postData(prebidEventsData *LogPrebidEvents) {
	data, err := proto.Marshal(prebidEventsData)
	if err != nil {
		if DebugLogging {
			fmt.Println(err)
		} // TODO : Log this
	} else {
		if DebugLogging {
			fmt.Println("Protobuf Marshal success!")
		}
		_, err = http.Post(newsIQDataEndpointDev, "", bytes.NewBuffer(data))

		if err != nil {
			if DebugLogging {
				fmt.Println(err)
			} // TODO : Log this
		} else {
			if DebugLogging {
				fmt.Println("Protobuf POST request success!")
			}
		}

		if DebugLogging {
			fmt.Println("RAW protobuf: ", data)
		}
	}
}

// Sends Auction Data to Collector
func (d *DataLogger) SendCollectorData(request *openrtb.BidRequest, response *openrtb.BidResponse, msg MsgType) {
	if DebugLogging {
		fmt.Println("News IQ Module - SendCollectorData() ", msg)
	}

	app := request.App
	site := request.Site

	var clientId uint64 = 0
	var clientDomain = ""

	if app != nil {
		if DebugLogging {
			fmt.Println("App type")
		}
		clientId, _ = strconv.ParseUint(app.ID, 10, 32)
		clientDomain = app.Domain
	} else if site != nil {
		if DebugLogging {
			fmt.Println("Site type")
		}
		clientId, _ = strconv.ParseUint(site.ID, 10, 32)
		clientDomain = site.Domain
	} else {
		if DebugLogging {
			fmt.Println("Client type key not found") // TODO : Log this
		}
	}

	adunitsArray := generateAdUnits(request, response)

	auctionObj := Auction{
		Version:              PrebidServerVersion,
		AuctionInitTimestamp: currentTimestamp(), // TODO : Update all timestamps
		PrebidAuctionId:      response.BidID,     // TODO : Is this correct?
		ConfiguredTimeoutMs:  30000,              // 30 seconds
		MsgType:              uint32(msg),
		AdUnits:              adunitsArray,
	}
	auctionsArray := []*Auction{&auctionObj}

	device := request.Device
	// device.DeviceType // TODO : Include this?
	deviceString := device.Make + " " + device.Model + " " + device.HWV + " " + device.OS + " " + device.OSV
	prebidEventObj := &LogPrebidEvents{
		Timestamp:       currentTimestamp(),
		RemoteAddrMacro: request.Device.IP,
		UserAgentMacro:  request.Device.UA,
		// RefererUrl:      clientDomain, // TODO : Should be a page url
		SellerMemberId: uint32(clientId),
		Domain:         clientDomain,
		Device:         deviceString,
		Auctions:       auctionsArray,
		NewsId:         request.ID,
	}

	postData(prebidEventObj)
}

//Module that can perform transactional logging
type DataLogger struct {
}

//Method to initialize the analytic module
func NewDataLogger(filename string) analytics.PBSAnalyticsModule {
	if DebugLogging {
		fmt.Println("News IQ Module - NewDataLogger - ", filename)
	}
	return &DataLogger{}
}

//Method to initialize the analytic module
func InitDataLogger() DataLogger {
	if DebugLogging {
		fmt.Println("News IQ Module - InitDataLogger")
	}
	return DataLogger{}
}

func currentTimestamp() uint64 {
	timeStamp := strings.ReplaceAll(time.Now().Format("01-02-2006"), "-", "")
	intTimeStamp, err := strconv.ParseUint(timeStamp, 10, 64)
	if err != nil {
		if DebugLogging {
			fmt.Println("Error: ", err)
		} // TODO: Log Error
		return 0
	}
	return intTimeStamp
}

func generateBidsArray(identifier string, response *openrtb.BidResponse) []*Bid {
	bidsArray := []*Bid{}
	seatBid := response.SeatBid
	if seatBid != nil {
		for _, seatbidObj := range seatBid {
			for _, bidObj := range seatbidObj.Bid {
				if bidObj.ID == identifier {
					creativeObj := &Creative{
						CreativeId: bidObj.CrID,
						Width:      uint32(bidObj.W),
						Height:     uint32(bidObj.H),
						Brand:      bidObj.Bundle,
					}

					bidResponseId := BidResponse
					if len(bidObj.NURL) > 0 { // Win
						bidResponseId = BidWon
					} else if len(bidObj.LURL) > 0 { // Lost
						bidResponseId = BidLost
					}
					bidTrackingObj := Bid{
						BidId:             fmt.Sprint(bidResponseId),
						Price:             bidObj.Price,
						BidderCode:        seatbidObj.Seat,
						BidderAdUnitId:    bidObj.ImpID,
						RequestTimestamp:  10212011,
						ResponseTimestamp: 02132011,
						// StatusCode:        uint32(ao.Status),  TODO : Bring this back in
						Source:   "",
						Creative: creativeObj,
					}
					bidsArray = append(bidsArray, &bidTrackingObj)
				}
			}
		}
	}
	return bidsArray
}

func generateAdUnits(request *openrtb.BidRequest, response *openrtb.BidResponse) []*AdUnit {

	adunitsArray := []*AdUnit{}
	for _, impressionObj := range request.Imp {
		paramsArray := []*Param{}
		if impressionObj.Banner != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "banner",
				}, &Param{
					Key:   "width",
					Value: fmt.Sprint(impressionObj.Banner.W),
				}, &Param{
					Key:   "height",
					Value: fmt.Sprint(impressionObj.Banner.H),
				}, &Param{
					Key:   "pos",
					Value: fmt.Sprint(impressionObj.Banner.Pos),
				}, &Param{
					Key:   "id",
					Value: fmt.Sprint(impressionObj.Banner.ID),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Video.Ext),
				})
		} else if impressionObj.Native != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "native",
				}, &Param{
					Key:   "request",
					Value: impressionObj.Native.Request,
				}, &Param{
					Key:   "version",
					Value: impressionObj.Native.Ver,
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Video.Ext),
				})
		} else if impressionObj.Video != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "video",
				}, &Param{
					Key:   "width",
					Value: fmt.Sprint(impressionObj.Video.W),
				}, &Param{
					Key:   "height",
					Value: fmt.Sprint(impressionObj.Video.H),
				}, &Param{
					Key:   "pos",
					Value: fmt.Sprint(impressionObj.Video.Pos),
				}, &Param{
					Key:   "min_duration",
					Value: fmt.Sprint(impressionObj.Video.MinDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Video.MaxDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Video.MaxDuration),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Video.Ext),
				})
		} else if impressionObj.Audio != nil {
			paramsArray = append(paramsArray,
				&Param{
					Key:   "adunit_type",
					Value: "audio",
				}, &Param{
					Key:   "min_duration",
					Value: fmt.Sprint(impressionObj.Video.MinDuration),
				}, &Param{
					Key:   "max_duration",
					Value: fmt.Sprint(impressionObj.Video.MaxDuration),
				}, &Param{
					Key:   "sequence",
					Value: fmt.Sprint(impressionObj.Audio.Sequence),
				}, &Param{
					Key:   "extension",
					Value: fmt.Sprint(impressionObj.Audio.Ext),
				})
		} else {
			if DebugLogging {
				fmt.Println("Impression object type missing") // TODO : Log this
			}
		}

		bidsArray := generateBidsArray(impressionObj.ID, response)
		adunitObj := AdUnit{
			AdUnitCode: impressionObj.ID,
			Bids:       bidsArray,
			Params:     paramsArray,
		}
		adunitsArray = append(adunitsArray, &adunitObj)
	}

	return adunitsArray
}
