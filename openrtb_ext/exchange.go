package openrtb_ext

import "github.com/mxmCherry/openrtb"

type Exchange struct {
	adapters []string
	bidRequest *openrtb.BidRequest
	cleanRequests map[string]*openrtb.BidRequest
}