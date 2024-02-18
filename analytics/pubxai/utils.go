package pubxai

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/docker/go-units"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

func NewBidQueue(queueType string, endpoint string, client *http.Client, clock clock.Clock, bufferInterval string, bufferSize string) *BidQueue {
	return &BidQueue{
		queue:          make([]Bid, 0),
		queueType:      queueType,
		endpoint:       endpoint,
		mutex:          sync.RWMutex{},
		httpClient:     client,
		clock:          clock,
		bufferSize:     bufferSize,
		bufferInterval: bufferInterval,
		lastSentTime:   clock.Now(),
	}
}

func (p *PubxaiModule) processLogData(ao *LogObject) {
	if ao == nil {
		return
	}
	if ao.RequestWrapper == nil {
		glog.Errorf("[pubxai] RequestWrapper is nil")
		return
	}
	request := ao.RequestWrapper.BidRequest
	imps := request.Imp
	var impsById = make(map[string]openrtb2.Imp)
	for _, imp := range imps {
		impsById[imp.ID] = imp
	}

	if len(imps) == 0 {
		glog.Errorf("[pubxai] No impressions in request")
		return
	}

	response := ao.Response
	if response == nil {
		glog.Errorf("[pubxai] Response is nil")
		return
	}
	// filter out bids not matching with impid
	seatBids := response.SeatBid
	if len(seatBids) == 0 {
		// TODO: check for NoBid case
		glog.Errorf("[pubxai] No seatbids in response")
		return
	}
	var BidResponses []map[string]interface{}
	for _, seatBid := range seatBids {
		bidderName := seatBid.Seat
		bids := seatBid.Bid
		for _, bid := range bids {
			if _, ok := impsById[bid.ImpID]; ok {
				temp := map[string]interface{}{
					"bidder": bidderName,
					"bid":    bid,
					"imp":    impsById[bid.ImpID],
				}
				BidResponses = append(BidResponses, temp)
			}
		}
	}
	glog.Infof("[pubxai] No of BidResponses: %v", len(BidResponses))
	if len(BidResponses) == 0 {
		glog.Errorf("[pubxai] No matching bids in response")
		return
	}
	p.processBidData(BidResponses, ao)
}

func (p *PubxaiModule) processBidData(bidResponses []map[string]interface{}, ao *LogObject) {
	startTime := ao.StartTime.UTC().UnixMilli()
	var requestExt map[string]interface{}
	var responseExt map[string]interface{}
	err := jsonutil.Unmarshal(ao.RequestWrapper.BidRequest.Ext, &requestExt)

	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling bidExt: %v", err)
		return
	}

	err = jsonutil.Unmarshal(ao.Response.Ext, &responseExt)
	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling bidExt: %v", err)
		return
	}

	floorData := requestExt["prebid"].(map[string]interface{})["floors"].(map[string]interface{})
	for _, bidData := range bidResponses {
		bidderName := bidData["bidder"].(string)
		bid := bidData["bid"].(openrtb2.Bid)
		imp := bidData["imp"].(openrtb2.Imp)
		var bidExt map[string]interface{}
		var impExt map[string]interface{}
		err := jsonutil.Unmarshal(imp.Ext, &impExt)
		if err != nil {
			glog.Errorf("[pubxai] Error unmarshalling impExt: %v", err)
			return
		}
		err = jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err != nil {
			glog.Errorf("[pubxai] Error unmarshalling bidExt: %v", err)
			return
		}
		bidderInfo := bidExt[bidderName].(map[string]interface{})
		bidderResponsetime := responseExt["responsetimemillis"].(map[string]interface{})[bidderName].(float64)
		bidObj := Bid{
			AdUnitCode:        bid.ImpID,
			BidId:             bid.ID,
			GptSlotCode:       "", //todo: get gptSlotCode
			AuctionId:         bidderInfo["auction_id"].(float64),
			BidderCode:        bidderInfo["bidder_id"].(float64),
			Cpm:               bidExt["origbidcpm"].(float64),
			CreativeId:        bid.CrID,
			Currency:          bidExt["origbidcur"].(string),
			FloorData:         floorData,
			NetRevenue:        true,
			RequestTimestamp:  startTime,
			ResponseTimestamp: startTime + int64(bidderResponsetime),
			Status:            "targetingSet",
			StatusMessage:     "Bid available",
			TimeToRespond:     int64(bidderResponsetime),
			TransactionId:     impExt["tid"].(string),
		}

		if p.isWinningBid(bidderName, bid, bidExt) {
			bidObj.IsWinningBid = true
			bidObj.RenderStatus = 4
			bidObj.PlacementId = impExt["prebid"].(map[string]interface{})["bidder"].(map[string]interface{})[bidderName].(map[string]interface{})["placement_id"].(float64)
			bidObj.RenderedSize = bidExt["prebid"].(map[string]interface{})["targeting"].(map[string]interface{})["hb_size"].(string)
			bidObj.FloorProvider = func() string {
				if val, ok := floorData["floorprovider"].(string); ok {
					return val
				}
				return ""
			}()

			bidObj.FloorFetchStatus = func() string {
				if val, ok := floorData["fetchstatus"].(string); ok {
					return val
				}
				return ""
			}()
			bidObj.FloorLocation = func() string {
				if val, ok := floorData["location"].(string); ok {
					return val
				}
				return ""
			}()

			bidObj.FloorModelVersion = func() string {
				if val, ok := floorData["modelversion"].(string); ok {
					return val
				}
				return ""
			}()

			bidObj.FloorSkipRate = func() int64 {
				if val, ok := floorData["skiprate"].(int64); ok {
					return val
				}
				return 0
			}()

			bidObj.IsFloorSkipped = func() bool {
				if val, ok := floorData["skipped"].(bool); ok {
					return val
				}
				return false
			}()
			glog.Infof("[pubxai] Enqueue to Winning bid Queue, Current Size: %v", len(p.winBidsQueue.queue))
			p.winBidsQueue.Enqueue(bidObj)
		} else {
			bidObj.IsWinningBid = false
			bidObj.RenderStatus = 0
			for _, format := range imp.Banner.Format {
				bidObj.Sizes = append(bidObj.Sizes, []int64{format.W, format.H})
			}
			p.auctionBidsQueue.Enqueue(bidObj)
		}

	}
}

// if hb_pb is present in bidExt.prebid.targeting and bidderName matches with hb_bidder
func (p *PubxaiModule) isWinningBid(bidderName string, bid openrtb2.Bid, bidExt map[string]interface{}) bool {
	if val, ok := bidExt["prebid"].(map[string]interface{})["targeting"].(map[string]interface{})["hb_pb"]; ok {
		if val != "" && bidderName == bidExt["prebid"].(map[string]interface{})["targeting"].(map[string]interface{})["hb_bidder"].(string) {
			return true
		}
	}
	return false
}

func (bidQueue *BidQueue) isTimeToSend() bool {
	timeDifference := bidQueue.clock.Since(bidQueue.lastSentTime)
	pDuration, err := time.ParseDuration(bidQueue.bufferInterval)
	if err != nil {
		glog.Errorf("[pubxai] Error parsing bufferInterval: %v", err)
		return false
	}
	glog.Infof("[pubxai] Time difference: %v, bufferInterval: %v", timeDifference, pDuration)
	return timeDifference >= pDuration
}

func (bidQueue *BidQueue) flushQueuedData() {

	if len(bidQueue.queue) == 0 {
		glog.Info("[pubxai] No queued data to send.")
	}

	data, err := jsonutil.Marshal(bidQueue.queue)
	if err != nil {
		glog.Errorf("[pubxai] Error marshaling event queue: %v", err)
	}

	resp, err := bidQueue.httpClient.Post(bidQueue.endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		glog.Errorf("[pubxai] Error sending queued data: %v", err)
		bidQueue.queue = nil
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		glog.Errorf("[pubxai] Unexpected response status: %s", resp.Status)
	} else {
		glog.Infof("[pubxai] Queued data sent successfully.")
	}
	// Clear the queue in any case
	bidQueue.queue = nil
}

func (bidQueue *BidQueue) Enqueue(item Bid) {
	bidQueue.mutex.Lock()
	defer bidQueue.mutex.Unlock()
	bidQueue.queue = append(bidQueue.queue, item)

	pBufferSize, err := units.FromHumanSize(bidQueue.bufferSize)
	if err != nil {
		glog.Errorf("[pubxai] Error parsing bufferSize: %v", err)
		return
	}
	if int64(len(bidQueue.queue)) >= pBufferSize || bidQueue.isTimeToSend() {
		bidQueue.flushQueuedData()
	}
}
