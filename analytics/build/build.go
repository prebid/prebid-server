package build

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
)

type EnabledAnalytics = analytics.EnabledAnalytics

func evaluateActivities(rw *openrtb_ext.RequestWrapper, ac privacy.ActivityControl, componentName string) (bool, *openrtb_ext.RequestWrapper) {
	return analytics.EvaluateActivities(rw, ac, componentName)
}

func updateReqWrapperForAnalytics(rw *openrtb_ext.RequestWrapper, adapterName string, isCloned bool) *openrtb_ext.RequestWrapper {
	return analytics.UpdateReqWrapperForAnalytics(rw, adapterName, isCloned)
}

func updatePrebidAnalyticsMap(extPrebidAnalytics map[string]json.RawMessage, adapterName string) map[string]json.RawMessage {
	return analytics.UpdatePrebidAnalyticsMap(extPrebidAnalytics, adapterName)
}
