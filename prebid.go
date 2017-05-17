package prebid

import (
	"context"

	"github.com/prebid/prebid-server/pbs"
)

type Adapter interface {
	Name() string
	FamilyName() string
	GetUsersyncInfo() *pbs.UsersyncInfo
	Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error)
}
