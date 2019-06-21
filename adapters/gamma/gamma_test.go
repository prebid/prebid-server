package gamma

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

type gBidInfo struct {
	deviceIP  string
	deviceUA  string
	deviceIFA string
	referrer  string
	width     uint64
	height    uint64
	tid       string
	buyerUID  string
	secure    bool
	currency  string
	delay     time.Duration
}

var gammaTestData gBidInfo

func equal(expected string, actual string, message string) (bool, *string) {
	if expected != actual {
		message := fmt.Sprintf("%s '%s' doesn't match '%s'", message, actual, expected)
		return false, &message
	}
	return true, nil
}

func assertGammaServerRequest(testData gBidInfo, r *http.Request, isOpenRtb bool) *string {
	if ok, err := equal("GET", r.Method, "HTTP method"); !ok {
		return err
	}
	if testData.secure {
		if ok, err := equal("https", r.URL.Scheme, "Scheme"); !ok {
			return err
		}
	}

	if ok, err := equal(testData.deviceUA, r.Header.Get("User-Agent"), "User agent"); !ok {
		return err
	}
	if ok, err := equal(testData.deviceIP, r.Header.Get("X-Forwarded-For"), "Device IP"); !ok {
		return err
	}
	if ok, err := equal(testData.referrer, r.Header.Get("Referer"), "Referer"); !ok {
		return err
	}
	return nil
}
func DummyGammaServer(w http.ResponseWriter, r *http.Request) {
	errorString := assertGammaServerRequest(gammaTestData, r, false)
	if errorString != nil {
		http.Error(w, *errorString, http.StatusInternalServerError)
		return
	}

	if gammaTestData.delay > 0 {
		<-time.After(gammaTestData.delay)
	}

	adformServerResponse, err := createGammaServerResponse(gammaTestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(adformServerResponse)
}
func createGammaServerResponse(testData gBidInfo) ([]byte, error) {
	// bids := []adformBid{
	// 	{
	// 		ResponseType: "banner",
	// 		Banner:       testData.tags[0].content,
	// 		Price:        testData.tags[0].price,
	// 		Currency:     "EUR",
	// 		Width:        testData.width,
	// 		Height:       testData.height,
	// 		DealId:       testData.tags[0].dealId,
	// 		CreativeId:   testData.tags[0].creativeId,
	// 	},
	// 	{},
	// 	{
	// 		ResponseType: "banner",
	// 		Banner:       testData.tags[2].content,
	// 		Price:        testData.tags[2].price,
	// 		Currency:     "EUR",
	// 		Width:        testData.width,
	// 		Height:       testData.height,
	// 		DealId:       testData.tags[2].dealId,
	// 		CreativeId:   testData.tags[2].creativeId,
	// 	},
	// }
	//gammaServerResponse, err := json.Marshal(bids)
	//return gammaServerResponse, err
	return nil, nil
}
func TestJsonSamples(t *testing.T) {
	fmt.Println("Start test")
	adapterstest.RunJSONBidderTest(t, "gammatest", NewGammaBidder("https://hb.gammaplatform.com/adx/request/"))
	fmt.Println("End  test")
}
