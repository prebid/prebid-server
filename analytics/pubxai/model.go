package pubxai

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

type Configuration struct {
	PublisherId        string `json:"publisher_id"`
	BufferInterval     string `json:"buffer_interval"`
	BufferSize         string `json:"buffer_size"`
	SamplingPercentage int    `json:"sampling_percentage"`
}

type PubxaiModule struct {
	publisherId      string
	endpoint         string
	winBidsQueue     *BidQueue
	auctionBidsQueue *BidQueue
	httpClient       *http.Client
	muxConfig        sync.RWMutex
	clock            clock.Clock
	cfg              *Configuration
	sigTermCh        chan os.Signal
	stopCh           chan struct{}
}

type BidQueue struct {
	queueType      string
	queue          []Bid
	bufferInterval string
	bufferSize     string
	endpoint       string
	httpClient     *http.Client
	lastSentTime   time.Time
	clock          clock.Clock
	mutex          sync.RWMutex
}

type Bid struct {
	AdUnitCode        string                 `json:"adUnitCode"`
	GptSlotCode       string                 `json:"gptSlotCode"`
	AuctionId         float64                `json:"auctionId"`
	BidderCode        float64                `json:"bidderCode"`
	Cpm               float64                `json:"cpm"`
	CreativeId        string                 `json:"creativeId"`
	Currency          string                 `json:"currency"`
	FloorData         map[string]interface{} `json:"floorData"`
	NetRevenue        bool                   `json:"netRevenue"`
	RequestTimestamp  int64                  `json:"requestTimestamp"`
	ResponseTimestamp int64                  `json:"responseTimestamp"`
	Status            string                 `json:"status"`
	StatusMessage     string                 `json:"statusMessage"`
	TimeToRespond     int64                  `json:"timeToRespond"`
	TransactionId     string                 `json:"transactionId"`
	BidId             string                 `json:"bidId"`
	RenderStatus      int64                  `json:"renderStatus"`
	Sizes             [][]int64              `json:"sizes"`
	FloorProvider     string                 `json:"floorProvider"`
	FloorFetchStatus  string                 `json:"floorFetchStatus"`
	FloorLocation     string                 `json:"floorLocation"`
	FloorModelVersion string                 `json:"floorModelVersion"`
	FloorSkipRate     int64                  `json:"floorSkipRate"`
	IsFloorSkipped    bool                   `json:"isFloorSkipped"`
	IsWinningBid      bool                   `json:"isWinningBid"`
	PlacementId       float64                `json:"placementId"`
	RenderedSize      string                 `json:"renderedSize"`
}
