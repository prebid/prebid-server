package identity

import (
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

const auctionContextKey = "intentiq.identity.auction"

type auctionContext struct {
	AuctionID        string
	Reference        string
	IP               string
	UserAgent        string
	ABTestUUID       string
	TerminationCause *int64
	Start            time.Time
	TraceEnabled     bool
	Trace            []string
}

func newAuctionContext(req *openrtb2.BidRequest) auctionContext {
	if req == nil {
		return auctionContext{}
	}
	result := auctionContext{AuctionID: req.ID}
	if req.Site != nil {
		result.Reference = req.Site.Domain
		if strings.TrimSpace(result.Reference) == "" {
			result.Reference = req.Site.Page
		}
	} else if req.App != nil {
		result.Reference = req.App.Bundle
		if strings.TrimSpace(result.Reference) == "" {
			result.Reference = req.App.Name
		}
	}
	if req.Device != nil {
		result.IP = req.Device.IP
		if result.IP == "" {
			result.IP = req.Device.IPv6
		}
		result.UserAgent = req.Device.UA
	}
	return result
}

func storeAuctionContext(value auctionContext) *hookstage.ModuleContext {
	ctx := hookstage.NewModuleContext()
	ctx.Set(auctionContextKey, value)
	return ctx
}

func loadAuctionContext(ctx *hookstage.ModuleContext) (auctionContext, bool) {
	if ctx == nil {
		return auctionContext{}, false
	}
	value, ok := ctx.Get(auctionContextKey)
	if !ok {
		return auctionContext{}, false
	}
	result, ok := value.(auctionContext)
	return result, ok
}
