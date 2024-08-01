package processor

import (
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	utils "github.com/prebid/prebid-server/v2/analytics/pubxai/utils"
)

type ProcessorService interface {
	ProcessLogData(ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid)
	ProcessBidData(bidResponses []map[string]interface{}, ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid)
}

type ProcessorServiceImpl struct {
	publisherId  string
	samplingRate int
	utilService  utils.UtilsService
}

func NewProcessorService(publisherId string, samplingRate int) ProcessorService {
	return &ProcessorServiceImpl{
		publisherId:  publisherId,
		samplingRate: samplingRate,
		utilService:  utils.NewUtilsService(publisherId),
	}
}

func (p *ProcessorServiceImpl) ProcessBidData(bidResponses []map[string]interface{}, ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid) {
	startTime := ao.StartTime.UTC().UnixMilli()

	requestExt, responseExt, err := p.utilService.UnmarshalExtensions(ao)
	if err != nil {
		glog.Errorf("[pubxai] Error unmarshalling extensions: %v", err)
		return nil, nil
	}

	auctionId := requestExt["id"].(string)

	adUnitCodes := p.utilService.ExtractAdunitCodes(requestExt)
	floorDetail := p.utilService.ExtractFloorDetail(requestExt, bidResponses[0])
	pageDetail := p.utilService.ExtractPageData(requestExt)
	deviceDetail := p.utilService.ExtractDeviceData(requestExt)
	userDetail := p.utilService.ExtractUserIds(requestExt)
	consentDetail := p.utilService.ExtractConsentTypes(requestExt)
	pmacDetail := map[string]interface{}{}

	auctionDetail := utils.AuctionDetail{
		AuctionId:   auctionId,
		RefreshRank: 0,
		Timestamp:   startTime,
		AdUnitCodes: adUnitCodes,
	}

	initOptions := utils.InitOptions{
		Pubxid:       p.publisherId,
		SamplingRate: p.samplingRate,
	}

	auctionBids, winningBids := p.utilService.ProcessBidResponses(bidResponses, auctionId, startTime, requestExt, responseExt, floorDetail)
	// make AuctionBids object and winningBids Object
	auctionObj := utils.AuctionBids{
		AuctionDetail: auctionDetail,
		FloorDetail:   floorDetail,
		PageDetail:    pageDetail,
		DeviceDetail:  deviceDetail,
		UserDetail:    userDetail,
		ConsentDetail: consentDetail,
		PmacDetail:    pmacDetail,
		InitOptions:   initOptions,
		Bids:          auctionBids,
		Source:        "PBS",
	}
	var winningBidsList []utils.WinningBid
	for _, winningBid := range winningBids {
		temp := utils.WinningBid{
			AuctionDetail: auctionDetail,
			FloorDetail:   floorDetail,
			PageDetail:    pageDetail,
			DeviceDetail:  deviceDetail,
			UserDetail:    userDetail,
			ConsentDetail: consentDetail,
			PmacDetail:    pmacDetail,
			InitOptions:   initOptions,
			WinningBid:    winningBid,
			Source:        "PBS",
		}
		winningBidsList = append(winningBidsList, temp)
	}
	return &auctionObj, winningBidsList

}

func (p *ProcessorServiceImpl) ProcessLogData(ao *utils.LogObject) (*utils.AuctionBids, []utils.WinningBid) {
	if ao == nil {
		return nil, nil
	}
	if ao.RequestWrapper == nil {
		glog.Errorf("[pubxai] RequestWrapper is nil")
		return nil, nil
	}
	request := ao.RequestWrapper.BidRequest
	imps := request.Imp
	var impsById = make(map[string]openrtb2.Imp)
	for _, imp := range imps {
		impsById[imp.ID] = imp
	}

	if len(imps) == 0 {
		glog.Errorf("[pubxai] No impressions in request")
		return nil, nil
	}

	response := ao.Response
	if response == nil {
		glog.Errorf("[pubxai] Response is nil")
		return nil, nil
	}
	seatBids := response.SeatBid
	if len(seatBids) == 0 {
		glog.Errorf("[pubxai] No seatbids in response")
		return nil, nil
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
		return nil, nil
	}
	return p.ProcessBidData(BidResponses, ao)
}
