package flipp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SirDataFR/iabtcfv2"
	"github.com/aws/smithy-go/ptr"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFlipp, config.Adapter{
		Endpoint: "http://example.com/pserver"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "flipptest", bidder)
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
				GDPR: ptr.Int8(1),
			},
		}
		result := paramsUserKeyPermitted(request)
		assert.New(t)
		assert.False(t, result, "param user key not permitted because Global Privacy Control is set")
	})
	t.Run("TCF purpose 4 is in scope and doesn't have consent", func(t *testing.T) {
		tcData := &iabtcfv2.TCData{
			CoreString: &iabtcfv2.CoreString{
				PublisherCC:       "test",
				Version:           2,
				Created:           time.Now(),
				LastUpdated:       time.Now(),
				CmpId:             92,
				CmpVersion:        1,
				ConsentScreen:     1,
				ConsentLanguage:   "EN",
				VendorListVersion: 32,
				TcfPolicyVersion:  2,
				PurposesConsent: map[int]bool{
					1: true,
					2: true,
					3: true,
				},
			},
		}
		segmentValue := tcData.CoreString.Encode()
		user := &openrtb2.User{
			Consent: segmentValue,
		}
		request := &openrtb2.BidRequest{
			User: user,
		}
		result := paramsUserKeyPermitted(request)
		assert.New(t)
		assert.False(t, result, "param user key not permitted because TCF purpose 4 is in scope and doesn't have consent")
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
