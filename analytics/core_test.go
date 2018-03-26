package analytics

import (
	"github.com/mxmCherry/openrtb"
	"net/http"
	"testing"
)

func TestNewPBSAnalytics(t *testing.T) {
	am := initAnalytics()
	am.LogAuctionObject(&AuctionObject{AUCTION, http.StatusOK, nil, &openrtb.BidRequest{}, &openrtb.BidResponse{}})
	if count != 1 {
		t.Errorf("PBSAnalyticsModule failed at LogAuctionObejct")
	}

	am.LogSetUIDObject(&SetUIDObject{SETUID, http.StatusOK, "bidders string", "uid", nil, true})
	if count != 2 {
		t.Errorf("PBSAnalyticsModule failed at LogSetUIDObejct")
	}

	am.LogCookieSyncObject(&CookieSyncObject{})
	if count != 3 {
		t.Errorf("PBSAnalyticsModule failed at LogCookieSyncObejct")
	}

	am.LogAmpObject(&AmpObject{})
	if count != 4 {
		t.Errorf("PBSAnalyticsModule failed at LogAmpObject")
	}
}

var count int = 0

type sampleModule struct{}

func (m *sampleModule) LogAuctionObject(ao *AuctionObject) { count++ }

func (m *sampleModule) LogCookieSyncObject(cso *CookieSyncObject) { count++ }

func (m *sampleModule) LogSetUIDObject(so *SetUIDObject) { count++ }

func (m *sampleModule) LogAmpObject(ao *AmpObject) { count++ }

func initAnalytics() PBSAnalyticsModule {
	modules := make(enabledAnalytics, 0)
	modules = append(modules, &sampleModule{})
	return &modules
}
