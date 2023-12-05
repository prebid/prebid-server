package http

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/util/randomutil"
)

type filterObjectFunc[T analytics.AuctionObject | analytics.AmpObject | analytics.CookieSyncObject | analytics.NotificationEvent | analytics.SetUIDObject | analytics.VideoObject] func(event *T) bool

func createFilter[T analytics.AuctionObject | analytics.AmpObject | analytics.VideoObject | analytics.SetUIDObject | analytics.CookieSyncObject | analytics.NotificationEvent](
	feature config.AnalyticsFeature,
	randomGenerator randomutil.RandomGenerator,
) (filterObjectFunc[T], error) {
	var filterProgram *vm.Program
	var err error
	if feature.Filter != "" {
		var obj T
		// precompile the filter expression for performance, make sure we return a boolean from the expression
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(obj), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *T) bool {
		// Disable tracking for nil events or events with a sample rate of 0
		if event == nil || feature.SampleRate <= 0 || randomGenerator.GenerateFloat64() > feature.SampleRate {
			return false
		}

		// Use a filter if one is defined
		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				glog.Errorf("[HttpAnalytics] Error filter: %v", err)
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}
