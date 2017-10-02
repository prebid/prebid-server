package decorators

import (
	"testing"
	"github.com/prebid/prebid-server/pbs"
	"context"
)

type brokenAdapter struct{}

func (a *brokenAdapter) Name() string {
	return "test"
}

// used for cookies and such
func (a *brokenAdapter) FamilyName() string {
	return "testFamily"
}

func (a *brokenAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return nil
}

func (a *brokenAdapter) SkipNoCookies() bool {
	return false
}

func (a *brokenAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (bids pbs.PBSBidSlice, err error) {
	panic("Fail!")
}

type workingAdapter struct{}

func (a *workingAdapter) Name() string {
	return "test"
}

// used for cookies and such
func (a *workingAdapter) FamilyName() string {
	return "testFamily"
}

func (a *workingAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return nil
}

func (a *workingAdapter) SkipNoCookies() bool {
	return false
}

func (a *workingAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (bids pbs.PBSBidSlice, err error) {
	bid := pbs.PBSBid{}
	return pbs.PBSBidSlice([]*pbs.PBSBid{&bid}), nil
}

func TestBrokenAdapter(t *testing.T) {
	safe := PreventPanics(&brokenAdapter{})
	bids, err := safe.Call(context.Background(), nil, nil)
	if bids != nil {
		t.Errorf("The wrapped adapter should return empty bids.")
	}
	if err == nil {
		t.Errorf("The wrapped adapter should return a non-nil error.")
	}
}

func TestWorkingAdapter(t *testing.T) {
	safe := PreventPanics(&workingAdapter{})
	bids, err := safe.Call(context.Background(), nil, nil)
	if len(bids) != 1 {
		t.Errorf("Working adapters should keep their bids.")
	}
	if err != nil {
		t.Errorf("Working adapters should still return a non-nil error.")
	}
}