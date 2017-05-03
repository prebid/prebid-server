package pbs

import "context"

type Adapter interface {
	Name() string
	FamilyName() string
	GetUsersyncInfo() *UsersyncInfo
	Call(ctx context.Context, req *PBSRequest, bidder *PBSBidder) (PBSBidSlice, error)
	SplitAdUnits() bool
}
