package dsa

import (
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// DSAWriter is used to write the default DSA to the request (req.regs.ext.dsa)
type DSAWriter struct {
	Config      *config.AccountDSA
	GDPRInScope bool
}

// Write sets the default DSA object on the request at regs.ext.dsa if it is
// defined in the account config and it is not already present on the request
func (dw DSAWriter) Write(req *openrtb_ext.RequestWrapper) (err error) {
	if req == nil {
		return
	}
	if getReqDSA(req) != nil {
		return
	}
	if dw.Config == nil || dw.Config.Default == nil {
		return
	}
	if dw.Config.GDPROnly && !dw.GDPRInScope {
		return
	}
	regExt, err := req.GetRegExt()
	if err != nil {
		return err
	}
	regExt.SetDSA(dw.Config.Default)
	return
}
