package rtdmodule

import (
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

type RtdProcessor interface {
	Process(request *openrtb2.BidRequest) []error
}

type RtdModule struct {
	name   string
	module func(request *openrtb2.BidRequest) error
}

type RtdModules struct {
	modules map[string]RtdModule
}

func (rtd *RtdModules) Process(request *openrtb2.BidRequest) []error {
	var rtdErrors []error
	for _, m := range rtd.modules {
		err := m.module(request)
		if err != nil {
			rtdErrors = append(rtdErrors, fmt.Errorf("Rtd module %s responser faild, error: %s", m.name, err.Error()))
		}
	}
	return rtdErrors
}
