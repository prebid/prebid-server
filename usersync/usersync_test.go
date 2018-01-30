package usersync

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/config"
)

func TestSyncers(t *testing.T) {
	cfg := &config.Configuration{}
	syncers := NewSyncerMap(cfg)
	for _, bidderName := range openrtb_ext.BidderMap {
		if _, ok := syncers[bidderName]; !ok {
			t.Errorf("No syncer exists for adapter: %s", bidderName)
		}
	}
}
