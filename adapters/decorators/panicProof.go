package decorators

import (
	"context"
	"fmt"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/pbs"
)

func PreventPanics(delegate adapters.Adapter) adapters.Adapter {
	return &panicProofAdapter{
		delegate: delegate,
	}
}

type panicProofAdapter struct {
	delegate adapters.Adapter
}

func (a *panicProofAdapter) Name() string {
	return a.delegate.Name()
}

// used for cookies and such
func (a *panicProofAdapter) FamilyName() string {
	return a.delegate.FamilyName()
}

func (a *panicProofAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.delegate.GetUsersyncInfo()
}

func (a *panicProofAdapter) SkipNoCookies() bool {
	return a.delegate.SkipNoCookies()
}

func (a *panicProofAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (bids pbs.PBSBidSlice, err error) {
	defer func() {
		if r := recover(); r != nil {
			bids = nil
			err = fmt.Errorf("Panic from bidder %s. %v", a.Name(), r)
		}
	}()
	return a.delegate.Call(ctx, req, bidder)
}
