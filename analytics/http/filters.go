package http

import (
	"math/rand"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
)

type (
	auctionFilter      func(event *analytics.AuctionObject) bool
	ampFilter          func(event *analytics.AmpObject) bool
	cookieSyncFilter   func(event *analytics.CookieSyncObject) bool
	notificationFilter func(event *analytics.NotificationEvent) bool
	setUIDFilter       func(event *analytics.SetUIDObject) bool
	videoFilter        func(event *analytics.VideoObject) bool
)

func createAuctionFilter(feature config.AnalyticsFeature) (auctionFilter, error) {
	var filterProgram *vm.Program
	var err error
	if feature.Filter != "" {
		// precompile the filter expression for performance, make sure we return a boolean from the expression
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.AuctionObject{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.AuctionObject) bool {
		// disbale tracking for nil events or events with a sample rate of 0
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		// Sample the event
		if rand.Float64() > feature.SampleRate {
			return false
		}

		// Use a filter is one is defined
		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}

func createAmpFilter(feature config.AnalyticsFeature) (ampFilter, error) {
	var filterProgram *vm.Program
	var err error

	if feature.Filter != "" {
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.AmpObject{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.AmpObject) bool {
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		if rand.Float64() > feature.SampleRate {
			return false
		}

		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}

func createCookieSyncFilter(feature config.AnalyticsFeature) (cookieSyncFilter, error) {
	var filterProgram *vm.Program
	var err error

	if feature.Filter != "" {
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.CookieSyncObject{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.CookieSyncObject) bool {
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		if rand.Float64() > feature.SampleRate {
			return false
		}

		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}

func createNotificationFilter(feature config.AnalyticsFeature) (notificationFilter, error) {
	var filterProgram *vm.Program
	var err error

	if feature.Filter != "" {
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.NotificationEvent{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.NotificationEvent) bool {
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		if rand.Float64() > feature.SampleRate {
			return false
		}

		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}

func createSetUIDFilter(feature config.AnalyticsFeature) (setUIDFilter, error) {
	var filterProgram *vm.Program
	var err error

	if feature.Filter != "" {
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.SetUIDObject{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.SetUIDObject) bool {
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		if rand.Float64() > feature.SampleRate {
			return false
		}

		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}

func createVideoFilter(feature config.AnalyticsFeature) (videoFilter, error) {
	var filterProgram *vm.Program
	var err error

	if feature.Filter != "" {
		filterProgram, err = expr.Compile(feature.Filter, expr.Env(analytics.VideoObject{}), expr.AsBool())
		if err != nil {
			return nil, err
		}
	}

	return func(event *analytics.VideoObject) bool {
		if event == nil || feature.SampleRate <= 0 {
			return false
		}

		if rand.Float64() > feature.SampleRate {
			return false
		}

		if filterProgram != nil {
			output, err := expr.Run(filterProgram, event)
			if err != nil {
				return false
			}
			return output.(bool)
		}

		return true
	}, nil
}
