package rulesengine

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func NewSetDevIp(params []string) Function {
	return &SetDeviceIp{IP: params[0]}
}

type SetDeviceIp struct {
	IP string
}

func (sdip *SetDeviceIp) Call(rw *openrtb_ext.RequestWrapper) (string, error) {
	// needs to create a mutation which captures the change set we want to apply
	// this function should not perform any modifications to the request
	// e.g. changeSet.AddMutation(mutation, hookstage.MutationUpdate, "bidrequest", "device", "ip")
	return "", nil
}

