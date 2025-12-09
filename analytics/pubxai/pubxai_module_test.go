package pubxai

import (
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	"github.com/prebid/prebid-server/v3/analytics"
	config "github.com/prebid/prebid-server/v3/analytics/pubxai/config"
	configMock "github.com/prebid/prebid-server/v3/analytics/pubxai/config/mocks"
	processorMock "github.com/prebid/prebid-server/v3/analytics/pubxai/processor/mocks"
	queue "github.com/prebid/prebid-server/v3/analytics/pubxai/queue"
	queueMock "github.com/prebid/prebid-server/v3/analytics/pubxai/queue/mocks"
	"github.com/prebid/prebid-server/v3/analytics/pubxai/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitializePubxAIModule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &http.Client{}
	mockClock := clock.NewMock()

	tests := []struct {
		name            string
		client          *http.Client
		publisherId     string
		endpoint        string
		bufferInterval  string
		bufferSize      string
		samplingPercent int
		configRefresh   string
		wantErr         bool
	}{
		{
			name:            "successful initialization",
			client:          mockClient,
			publisherId:     "testPublisher",
			endpoint:        "http://test-endpoint",
			bufferInterval:  "10s",
			bufferSize:      "1MB",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         false,
		},
		{
			name:            "client is nil",
			client:          nil,
			publisherId:     "testPublisher",
			endpoint:        "http://test-endpoint",
			bufferInterval:  "10s",
			bufferSize:      "1MB",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         true,
		},
		{
			name:            "empty publisherId",
			client:          mockClient,
			publisherId:     "",
			endpoint:        "http://test-endpoint",
			bufferInterval:  "10s",
			bufferSize:      "1MB",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         true,
		},
		{
			name:            "empty endpoint",
			client:          mockClient,
			publisherId:     "testPublisher",
			endpoint:        "",
			bufferInterval:  "10s",
			bufferSize:      "1MB",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         true,
		},
		{
			name:            "invalid bufferInterval",
			client:          mockClient,
			publisherId:     "testPublisher",
			endpoint:        "http://test-endpoint",
			bufferInterval:  "invalid",
			bufferSize:      "1MB",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         true,
		},
		{
			name:            "invalid bufferSize",
			client:          mockClient,
			publisherId:     "testPublisher",
			endpoint:        "http://test-endpoint",
			bufferInterval:  "10s",
			bufferSize:      "invalid",
			samplingPercent: 50,
			configRefresh:   "1m",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, err := InitializePubxAIModule(tt.client, tt.publisherId, tt.endpoint, tt.bufferInterval, tt.bufferSize, tt.samplingPercent, tt.configRefresh, mockClock)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, module)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, module)
			}
		})
	}
}

func TestPubxaiModule_start(t *testing.T) {
	tests := []struct {
		name          string
		initialConfig *config.Configuration
		newConfig     *config.Configuration
		configIsSame  bool
		expectUpdate  bool
	}{
		{
			name: "should update config when received from channel",
			initialConfig: &config.Configuration{
				PublisherId:    "initial",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			newConfig: &config.Configuration{
				PublisherId:    "updated",
				BufferInterval: "20s",
				BufferSize:     "2MB",
			},
			configIsSame: false,
			expectUpdate: true,
		},
		{
			name: "should ignore config update if config is same",
			initialConfig: &config.Configuration{
				PublisherId:    "same",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			newConfig: &config.Configuration{
				PublisherId:    "same",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			configIsSame: true,
			expectUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockConfigService := &configMock.MockConfigService{}
			mockAuctionBidsQueue := queueMock.NewMockQueueService[utils.AuctionBids](t)
			mockWinBidsQueue := queueMock.NewMockQueueService[utils.WinningBid](t)

			// Create channels
			configChan := make(chan *config.Configuration)
			sigTermChan := make(chan os.Signal)
			stopChan := make(chan struct{})
			done := make(chan struct{})

			p := &PubxaiModule{
				sigTermCh:     sigTermChan,
				stopCh:        stopChan,
				cfg:           tt.initialConfig,
				configService: mockConfigService,
				auctionBidsQueue: &queue.AuctionBidsQueue{
					QueueService: mockAuctionBidsQueue,
				},
				winBidsQueue: &queue.WinningBidQueue{
					QueueService: mockWinBidsQueue,
				},
			}

			// Set up mock expectations
			mockConfigService.On("IsSameAs", tt.newConfig, tt.initialConfig).Return(tt.configIsSame).Run(func(args mock.Arguments) {
				if !tt.configIsSame {
					// For updates, wait until after UpdateConfig is called
					return
				}
				done <- struct{}{}
			})

			if tt.expectUpdate {
				mockAuctionBidsQueue.On("UpdateConfig", tt.newConfig.BufferInterval, tt.newConfig.BufferSize).Return(nil).Run(func(args mock.Arguments) {
					done <- struct{}{}
				})
				mockWinBidsQueue.On("UpdateConfig", tt.newConfig.BufferInterval, tt.newConfig.BufferSize).Return(nil).Run(func(args mock.Arguments) {
					done <- struct{}{}
				})
			}

			go p.start(configChan)
			configChan <- tt.newConfig

			select {
			case <-done:
				if tt.expectUpdate {
					assert.Equal(t, tt.newConfig, p.cfg)
				}
			case <-time.After(time.Second):
				t.Fatal("timeout waiting for config update")
			}

			close(sigTermChan)
			mockConfigService.AssertExpectations(t)
			mockAuctionBidsQueue.AssertExpectations(t)
			mockWinBidsQueue.AssertExpectations(t)
		})
	}
}

func TestPubxaiModule_updateConfig(t *testing.T) {
	tests := []struct {
		name          string
		initialConfig *config.Configuration
		newConfig     *config.Configuration
		configIsSame  bool
	}{
		{
			name: "config is the same",
			initialConfig: &config.Configuration{
				PublisherId:    "samePublisher",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			newConfig: &config.Configuration{
				PublisherId:    "samePublisher",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			configIsSame: true,
		},
		{
			name: "config is different",
			initialConfig: &config.Configuration{
				PublisherId:    "oldPublisher",
				BufferInterval: "10s",
				BufferSize:     "1MB",
			},
			newConfig: &config.Configuration{
				PublisherId:    "newPublisher",
				BufferInterval: "20s",
				BufferSize:     "2MB",
			},
			configIsSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock dependencies using testify
			mockConfigService := &configMock.MockConfigService{}
			mockAuctionBidsQueue := queueMock.NewMockQueueService[utils.AuctionBids](t)
			mockWinBidsQueue := queueMock.NewMockQueueService[utils.WinningBid](t)

			p := &PubxaiModule{
				cfg:           tt.initialConfig,
				configService: mockConfigService,
				auctionBidsQueue: &queue.AuctionBidsQueue{
					QueueService: mockAuctionBidsQueue,
				},
				winBidsQueue: &queue.WinningBidQueue{
					QueueService: mockWinBidsQueue,
				},
				muxConfig: sync.RWMutex{},
			}

			// Set expectations
			mockConfigService.On("IsSameAs", tt.newConfig, tt.initialConfig).Return(tt.configIsSame)

			if !tt.configIsSame {
				mockAuctionBidsQueue.On("UpdateConfig", tt.newConfig.BufferInterval, tt.newConfig.BufferSize).Return(nil)
				mockWinBidsQueue.On("UpdateConfig", tt.newConfig.BufferInterval, tt.newConfig.BufferSize).Return(nil)
			}

			p.updateConfig(tt.newConfig)

			if !tt.configIsSame {
				assert.Equal(t, tt.newConfig, p.cfg)
			} else {
				assert.Equal(t, tt.initialConfig, p.cfg)
			}
			mockConfigService.AssertExpectations(t)
		})
	}
}

func TestPubxaiModule_pushToQueue(t *testing.T) {
	// Test data setup
	auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
	winningBids := []utils.WinningBid{
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
	}

	tests := []struct {
		name          string
		auctionBids   *utils.AuctionBids
		winningBids   []utils.WinningBid
		expectAuction bool
		expectWins    bool
	}{
		{
			name:          "only winning bids",
			auctionBids:   nil,
			winningBids:   winningBids,
			expectAuction: false,
			expectWins:    true,
		},
		{
			name:          "only auction bids",
			auctionBids:   auctionBids,
			winningBids:   nil,
			expectAuction: true,
			expectWins:    false,
		},
		{
			name:          "both auction bids and winning bids",
			auctionBids:   auctionBids,
			winningBids:   winningBids,
			expectAuction: true,
			expectWins:    true,
		},
		{
			name:          "neither auction bids nor winning bids",
			auctionBids:   nil,
			winningBids:   nil,
			expectAuction: false,
			expectWins:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock dependencies
			mockAuctionBidsQueue := queueMock.NewMockQueueService[utils.AuctionBids](t)
			mockWinBidsQueue := queueMock.NewMockQueueService[utils.WinningBid](t)

			p := &PubxaiModule{
				auctionBidsQueue: &queue.AuctionBidsQueue{
					QueueService: mockAuctionBidsQueue,
				},
				winBidsQueue: &queue.WinningBidQueue{
					QueueService: mockWinBidsQueue,
				},
			}

			// Set expectations based
			if tt.expectAuction {
				mockAuctionBidsQueue.On("Enqueue", *auctionBids).Return(nil)
			}

			if tt.expectWins {
				for _, win := range winningBids {
					mockWinBidsQueue.On("Enqueue", win).Return(nil)
				}
			}

			// Execute test
			p.pushToQueue(tt.auctionBids, tt.winningBids)

			// Verify expectations
			mockAuctionBidsQueue.AssertExpectations(t)
			mockWinBidsQueue.AssertExpectations(t)
		})
	}
}

func TestPubxaiModule_LogAuctionObject(t *testing.T) {
	mockProcessorService := processorMock.NewMockProcessorService()
	mockAuctionBidsQueue := queueMock.NewMockQueueService[utils.AuctionBids](t)
	mockWinBidsQueue := queueMock.NewMockQueueService[utils.WinningBid](t)

	auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
	winningBids := []utils.WinningBid{
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
	}

	tests := []struct {
		name             string
		samplingPercent  int
		auctionObj       *analytics.AuctionObject
		expectProcessing bool
	}{
		{
			name:             "Empty AuctionObject",
			samplingPercent:  0,
			auctionObj:       nil,
			expectProcessing: false,
		},
		{
			name:             "Sampling percentage too low",
			samplingPercent:  0,
			auctionObj:       &analytics.AuctionObject{Status: 0},
			expectProcessing: false,
		},
		{
			name:            "Sampling percentage allows logging",
			samplingPercent: 100,
			auctionObj: &analytics.AuctionObject{
				Status:    0,
				StartTime: time.Now(),
			},
			expectProcessing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations for each test case
			mockProcessorService.Mock = mock.Mock{}

			p := &PubxaiModule{
				processorService: mockProcessorService,
				auctionBidsQueue: &queue.AuctionBidsQueue{
					QueueService: mockAuctionBidsQueue,
				},
				winBidsQueue: &queue.WinningBidQueue{
					QueueService: mockWinBidsQueue,
				},
				cfg: &config.Configuration{SamplingPercentage: tt.samplingPercent},
			}

			if tt.expectProcessing {
				mockProcessorService.On("ProcessLogData", mock.Anything).Return(auctionBids, winningBids)
				mockAuctionBidsQueue.On("Enqueue", *auctionBids).Return(nil)
				mockWinBidsQueue.On("Enqueue", winningBids[0]).Return(nil)
				mockWinBidsQueue.On("Enqueue", winningBids[1]).Return(nil)
			}

			p.LogAuctionObject(tt.auctionObj)

			// Assert that all expectations were met
			mockProcessorService.AssertExpectations(t)
			mockAuctionBidsQueue.AssertExpectations(t)
			mockWinBidsQueue.AssertExpectations(t)
		})
	}
}
