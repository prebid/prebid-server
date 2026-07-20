package tracker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
)

// Inject TrackerCall in Native Adm
func injectNativeCreativeTrackers(native *openrtb2.Native, bid openrtb2.Bid, tracker models.OWTracker, endpoint string) (string, string, error) {
	adm := bid.AdM
	var err error
	if sdkutils.IsSdkBiddingEndpoint(endpoint) {
		return adm, getBURL(bid.BURL, tracker), nil
	}
	if native == nil {
		return adm, bid.BURL, errors.New("native object is missing")
	}
	if len(native.Request) == 0 {
		return adm, bid.BURL, errors.New("native request is empty")
	}
	setTrackerURL := false
	callback := func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}
		if setTrackerURL {
			return
		}
		adm, setTrackerURL = injectNativeEventTracker(&adm, value, tracker)
	}
	jsonparser.ArrayEach([]byte(native.Request), callback, models.EventTrackers)

	if setTrackerURL {
		return adm, bid.BURL, nil
	}
	adm, err = injectNativeImpressionTracker(&adm, tracker)
	return adm, bid.BURL, err
}

// inject tracker in EventTracker Object
func injectNativeEventTracker(adm *string, value []byte, trackerParam models.OWTracker) (string, bool) {
	//Check for event=1
	event, _, _, err := jsonparser.Get(value, models.Event)
	if err != nil || string(event) != models.EventValue {
		return *adm, false
	}
	//Check for method=1
	methodsArray, _, _, err := jsonparser.Get(value, models.Methods) // "[1]","[2]","[1,2]", "[2,1]"
	if err != nil || !strings.Contains(string(methodsArray), models.MethodValue) {
		return *adm, false
	}

	nativeEventTracker := strings.Replace(models.NativeTrackerMacro, "${trackerUrl}", trackerParam.TrackerURL, 1)
	newAdm, err := jsonparser.Set([]byte(*adm), []byte(nativeEventTracker), models.EventTrackers, "[]")
	if err != nil {
		return *adm, false
	}
	*adm = string(newAdm)
	return *adm, true
}

// inject tracker in ImpTracker Object
func injectNativeImpressionTracker(adm *string, tracker models.OWTracker) (string, error) {
	impTrackers := []string{}
	callback := func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		impTrackers = append(impTrackers, string(value))
	}
	jsonparser.ArrayEach([]byte(*adm), callback, models.ImpTrackers)
	//append trackerUrl
	impTrackers = append(impTrackers, tracker.TrackerURL)
	allImpTrackers := fmt.Sprintf(`["%s"]`, strings.Join(impTrackers, `","`))
	newAdm, err := jsonparser.Set([]byte(*adm), []byte(allImpTrackers), models.ImpTrackers)
	if err != nil {
		return *adm, errors.New("error setting imptrackers in native adm")
	}
	*adm = string(newAdm)
	return *adm, nil
}
