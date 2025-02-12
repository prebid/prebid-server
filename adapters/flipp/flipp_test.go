package flipp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const fakeUuid = "30470a14-2949-4110-abce-b62d57304ad5"

type TestUUIDGenerator struct{}

func (TestUUIDGenerator) Generate() (string, error) {
	return fakeUuid, nil
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFlipp, config.Adapter{
		Endpoint: "http://example.com/pserver"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	setFakeUUIDGenerator(bidder)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "flipptest", bidder)
}

func setFakeUUIDGenerator(bidder adapters.Bidder) {
	bidderFlipp, _ := bidder.(*adapter)
	bidderFlipp.uuidGenerator = TestUUIDGenerator{}
}

func TestParamsUserKeyPermitted(t *testing.T) {
	t.Run("Coppa is in effect", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			Regs: &openrtb2.Regs{
				COPPA: 1,
			},
		}
		result := paramsUserKeyPermitted(request)
		assert.New(t)
		assert.False(t, result, "param user key not permitted because coppa is in effect")
	})
	t.Run("The Global Privacy Control is set", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			Regs: &openrtb2.Regs{
				GDPR: openrtb2.Int8Ptr(1),
			},
		}
		result := paramsUserKeyPermitted(request)
		assert.New(t)
		assert.False(t, result, "param user key not permitted because Global Privacy Control is set")
	})
	t.Run("The Prebid transmitEids activity is disallowed", func(t *testing.T) {
		extData := struct {
			TransmitEids bool `json:"transmitEids"`
		}{
			TransmitEids: false,
		}
		ext, err := json.Marshal(extData)
		if err != nil {
			t.Fatalf("failed to marshal ext data: %v", err)
		}
		request := &openrtb2.BidRequest{
			Ext: ext,
		}

		result := paramsUserKeyPermitted(request)
		assert.New(t)
		assert.False(t, result, "param user key not permitted because Prebid transmitEids activity is disallowed")
	})
}
