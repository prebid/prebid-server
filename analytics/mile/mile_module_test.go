package mile

import (
	"fmt"
	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/clients"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEventSend(t *testing.T) {
	config := Configuration{
		ScopeID:  "test",
		Endpoint: "http://localhost:8000",
		Features: map[string]bool{
			auction:    true,
			video:      true,
			amp:        true,
			cookieSync: true,
			setUID:     true,
		},
	}

	module, err := NewModuleWithConfig(clients.GetDefaultHttpInstance(),
		"test",
		"http://localhost:8000/pageview-event/json",
		&config,
		1,
		"100",
		"5s",
		clock.New(),
	)
	assert.NoError(t, err)

	fmt.Println(module)
	//fmt.Println

	analyticsEvent := analytics.AuctionObject{}
	module.LogAuctionObject(&analyticsEvent)
	module.LogAuctionObject(&analyticsEvent)

	time.Sleep(10 * time.Second)

	//assert.ElementsMatch(t, []byte{'1', '2', '3'}, []byte(readGz(data)))
}
