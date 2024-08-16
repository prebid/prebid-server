package processor

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/openrtb2"
	utils "github.com/prebid/prebid-server/v2/analytics/pubxai/utils"
	utilsMock "github.com/prebid/prebid-server/v2/analytics/pubxai/utils/mocks"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestProcessLogData_NilAuctionObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := (*utils.LogObject)(nil)
	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.Nil(t, auctionBids)
	assert.Nil(t, winningBids)
}

func TestProcessLogData_NilRequestWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{}
	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.Nil(t, auctionBids)
	assert.Nil(t, winningBids)
}

func TestProcessLogData_NoImpressions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{},
		},
	}
	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.Nil(t, auctionBids)
	assert.Nil(t, winningBids)
}

func TestProcessLogData_NilResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "imp1"}},
			},
		},
		Response: nil,
	}
	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.Nil(t, auctionBids)
	assert.Nil(t, winningBids)
}

func TestProcessLogData_UnmarshalExtensionsFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "imp1"}},
			},
		},
		Response: &openrtb2.BidResponse{
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "bidder1",
					Bid:  []openrtb2.Bid{{ImpID: "imp1"}},
				},
			},
		},
		StartTime: time.Now(),
	}

	mockUtilsService.EXPECT().UnmarshalExtensions(ao).Return(nil, nil, errors.New("Invalid Data"))

	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.Nil(t, auctionBids)
	assert.Nil(t, winningBids)
}

func TestProcessLogData_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "imp1"}},
			},
		},
		Response: &openrtb2.BidResponse{
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "bidder1",
					Bid:  []openrtb2.Bid{{ImpID: "imp1"}},
				},
			},
		},
		StartTime: time.Now(),
	}
	wBids := []utils.Bid{
		{BidId: "bidder1"},
	}

	mockUtilsService.EXPECT().UnmarshalExtensions(ao).Return(map[string]interface{}{"id": "auctionId"}, map[string]interface{}{}, nil)
	mockUtilsService.EXPECT().ExtractAdunitCodes(gomock.Any()).Return([]string{"adUnitCode"})
	mockUtilsService.EXPECT().ExtractFloorDetail(gomock.Any()).Return(utils.FloorDetail{})
	mockUtilsService.EXPECT().ExtractPageData(gomock.Any()).Return(utils.PageDetail{})
	mockUtilsService.EXPECT().ExtractDeviceData(gomock.Any()).Return(utils.DeviceDetail{})
	mockUtilsService.EXPECT().ExtractUserIds(gomock.Any()).Return(utils.UserDetail{})
	mockUtilsService.EXPECT().ExtractConsentTypes(gomock.Any()).Return(utils.ConsentDetail{})
	mockUtilsService.EXPECT().ProcessBidResponses(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]utils.Bid{}, wBids)
	mockUtilsService.EXPECT().AppendTimeoutBids(gomock.Any(), gomock.Any(), gomock.Any()).Return([]utils.Bid{})
	auctionBids, winningBids := processorService.ProcessLogData(ao)
	assert.NotNil(t, auctionBids)
	assert.NotNil(t, winningBids)
}

func TestProcessBidData_Success_NoWinningBid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUtilsService := utilsMock.NewMockUtilsService(ctrl)
	processorService := &ProcessorServiceImpl{
		utilService: mockUtilsService,
	}

	ao := &utils.LogObject{
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "imp1"}},
			},
		},
		Response: &openrtb2.BidResponse{
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "bidder1",
					Bid:  []openrtb2.Bid{{ImpID: "imp1"}},
				},
			},
		},
		StartTime: time.Now(),
	}
	bidResponses := []map[string]interface{}{
		{
			"bidder": "bidder1",
			"bid":    openrtb2.Bid{ImpID: "imp1"},
			"imp":    openrtb2.Imp{ID: "imp1"},
		},
	}
	request := ao.RequestWrapper.BidRequest
	var impsById = make(map[string]openrtb2.Imp)
	imps := request.Imp
	for _, imp := range imps {
		impsById[imp.ID] = imp
	}
	mockUtilsService.EXPECT().UnmarshalExtensions(ao).Return(map[string]interface{}{"id": "auctionId"}, map[string]interface{}{}, nil)
	mockUtilsService.EXPECT().ExtractAdunitCodes(gomock.Any()).Return([]string{"adUnitCode"})
	mockUtilsService.EXPECT().ExtractFloorDetail(gomock.Any()).Return(utils.FloorDetail{})
	mockUtilsService.EXPECT().ExtractPageData(gomock.Any()).Return(utils.PageDetail{})
	mockUtilsService.EXPECT().ExtractDeviceData(gomock.Any()).Return(utils.DeviceDetail{})
	mockUtilsService.EXPECT().ExtractUserIds(gomock.Any()).Return(utils.UserDetail{})
	mockUtilsService.EXPECT().ExtractConsentTypes(gomock.Any()).Return(utils.ConsentDetail{})
	mockUtilsService.EXPECT().ProcessBidResponses(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]utils.Bid{}, []utils.Bid{})
	mockUtilsService.EXPECT().AppendTimeoutBids(gomock.Any(), gomock.Any(), gomock.Any()).Return([]utils.Bid{})

	auctionBids, winningBids := processorService.ProcessBidData(bidResponses, impsById, ao)
	assert.NotNil(t, auctionBids)
	assert.Nil(t, winningBids)
}
