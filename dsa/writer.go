package dsa

import (
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Writer is used to write the default DSA to the request (req.regs.ext.dsa)
type Writer struct {
	Config      *config.AccountDSA
	GDPRInScope bool
}

// Write sets the default DSA object on the request at regs.ext.dsa if it is
// defined in the account config and it is not already present on the request
func (dw Writer) Write(req *openrtb_ext.RequestWrapper) error {
	if req == nil || getReqDSA(req) != nil {
		return nil
	}
	if dw.Config == nil || dw.Config.DefaultUnpacked == nil {
		return nil
	}
	if dw.Config.GDPROnly && !dw.GDPRInScope {
		return nil
	}
	regExt, err := req.GetRegExt()
	if err != nil {
		return err
	}
	clonedDefaultUnpacked := dw.Config.DefaultUnpacked.Clone()
	regExt.SetDSA(clonedDefaultUnpacked)
	return nil
}
