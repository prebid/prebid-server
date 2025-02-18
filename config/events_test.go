package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidCreateElement(t *testing.T) {
	testCases := []struct {
		description   string
		createElement VASTEventElement
		valid         bool
	}{
		{
			description:   "Empty create element",
			createElement: "",
			valid:         false,
		},
		{
			description:   "Invalid create element",
			createElement: "invalid_element",
			valid:         false,
		},
		{
			description:   "Valid create element",
			createElement: "tracking",
			valid:         true,
		},
		{
			description:   "Case sensitivity of create element",
			createElement: "Tracking",
			valid:         false,
		},
	}
	for _, test := range testCases {
		isValid := test.createElement.isValid()
		assert.Equal(t, test.valid, isValid, test.description)
	}

}

func TestIsValidTrackingEventType(t *testing.T) {
	testCases := []struct {
		description string
		vastEvent   VASTEvent
		valid       bool
	}{
		{
			description: "Empty type",
			vastEvent:   VASTEvent{},
			valid:       false,
		},
		{
			description: "Empty type for tracking event",
			vastEvent: VASTEvent{
				CreateElement: TrackingVASTElement,
			},
			valid: false,
		},
		{
			description: "Invalid type for tracking event",
			vastEvent: VASTEvent{
				CreateElement: TrackingVASTElement,
				Type:          "invalid_type",
			},
			valid: false,
		},
		{
			description: "Valid type for tracking event",
			vastEvent: VASTEvent{
				CreateElement: TrackingVASTElement,
				Type:          MidPoint,
			},
			valid: true,
		},
		{
			description: "Case sensitivity of type for tracking event",
			vastEvent: VASTEvent{
				CreateElement: TrackingVASTElement,
				Type:          "COMplete",
			},
			valid: false,
		},
	}
	for _, test := range testCases {
		isValid := test.vastEvent.Type.isValid()
		assert.Equal(t, test.valid, isValid, test.description)
	}
}

func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		description string
		url         string
		valid       bool
	}{
		{
			description: "Empty Url",
			url:         "",
			valid:       false,
		},
		{
			description: "Capital Domain name",
			url:         "http://PREBID.ORG",
			valid:       true,
		},
		{
			description: "Url with Macros",
			url:         "https://PBS_HOST/event?t=##PBS-EVENTTYPE##&vtype=##PBS-VASTEVENT##&b=##PBS-BIDID##&f=i&a=##PBS-ACCOUNTID##&ts=##PBS-TIMESTAMP##&bidder=##PBS-BIDDER##&int=##PBS-INTEGRATION##&mt=##PBS-MEDIATYPE##&ch=##PBS-CHANNEL##&aid=##PBS-AUCTIONID##&l=##PBS-LINEID##",
			valid:       true,
		},
		{
			description: "Invalid http syntax",
			url:         "HTTP://PREBID.ORG",
			valid:       false,
		},
		{
			description: "Invalid protocol",
			url:         "HTTPE://PREBID.ORG",
			valid:       false,
		},
		{
			description: "Double colon after domain name",
			url:         "https://prebid.org::8000",
			valid:       false,
		},
		{
			description: "Url with Port",
			url:         "https://prebid.org:8000",
			valid:       true,
		},
		{
			description: "Url with invalid Port",
			url:         "https://prebid.org:100000",
			valid:       false,
		},
	}
	for _, test := range testCases {
		isValid := isValidURL(test.url)
		assert.Equal(t, test.valid, isValid, test.description)
	}
}

func TestIsTrackingEvent(t *testing.T) {
	testCases := []struct {
		description string
		vastEvent   VASTEvent
		valid       bool
	}{
		{
			description: "Empty Tracking Event",
			vastEvent:   VASTEvent{},
			valid:       false,
		},
		{
			description: "Valid Tracking Event",
			vastEvent: VASTEvent{
				CreateElement: "tracking",
			},
			valid: true,
		},
		{
			description: "Case Sensitivity in Tracking Event",
			vastEvent: VASTEvent{
				CreateElement: "Tracking",
			},
			valid: false,
		},
	}
	for _, test := range testCases {
		isValid := test.vastEvent.isTrackingEvent()
		assert.Equal(t, test.valid, isValid, test.description)
	}
}

func TestValidateVASTEvent(t *testing.T) {
	testCases := []struct {
		description string
		vastEvent   VASTEvent
		index       int
		expectErr   bool
	}{
		{
			description: "At least one URL - Default URL",
			vastEvent: VASTEvent{
				CreateElement:     "tracking",
				Type:              "start",
				ExcludeDefaultURL: false,
				URLs:              nil,
			},
			index:     0,
			expectErr: false,
		},
		{
			description: "URLs has at least one URL, default URL is excluded",
			vastEvent: VASTEvent{
				CreateElement:     "impression",
				ExcludeDefaultURL: true,
				URLs:              []string{"http://mytracker.comm"},
			},
			index:     0,
			expectErr: false,
		},
		{
			description: "No URLs and default URL Excluded",
			vastEvent: VASTEvent{
				CreateElement:     "impression",
				ExcludeDefaultURL: true,
				URLs:              nil,
			},
			index:     0,
			expectErr: true,
		},
		{
			description: "Invalid Create Elemment",
			vastEvent:   VASTEvent{},
			index:       0,
			expectErr:   true,
		},
		{
			description: "Invalid Type",
			vastEvent: VASTEvent{
				CreateElement: "tracking",
				Type:          "invalid_type",
			},
			index:     0,
			expectErr: true,
		},
		{
			description: "Invalid URLs",
			vastEvent: VASTEvent{
				CreateElement: "impression",
				URLs:          []string{"http://url.com?k1=v1&k2=###PBS-MACRO##", "httpE://invalid.url"},
			},
			index:     0,
			expectErr: true,
		},
		{
			description: "Valid type but create element is other than tracking",
			vastEvent: VASTEvent{
				CreateElement: "impression",
				Type:          "start",
			},
			index:     0,
			expectErr: true,
		},
	}
	for _, test := range testCases {
		err := test.vastEvent.validate()
		assert.Equal(t, !test.expectErr, err == nil, test.description)
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		description string
		events      Events
		expectErr   bool
	}{
		{
			description: "Empty default URL",
			events: Events{
				Enabled: true,
			},
			expectErr: true,
		},
		{
			description: "Events are disabled. Skips validations",
			events: Events{
				Enabled:    false,
				DefaultURL: "",
			},
			expectErr: false,
		},
		{
			description: "No VAST Events and default URL present",
			events: Events{
				Enabled:    true,
				DefaultURL: "http://prebid.org",
			},
			expectErr: false,
		},
		{
			description: "Invalid VAST Event",
			events: Events{
				Enabled:    true,
				DefaultURL: "http://prebid.org",
				VASTEvents: []VASTEvent{
					{},
				},
			},
			expectErr: true,
		},
	}
	for _, test := range testCases {
		errs := test.events.validate(make([]error, 0))
		assert.Equal(t, !test.expectErr, len(errs) == 0, test.description)
	}
}

func TestValidateVASTEvents(t *testing.T) {
	testCases := []struct {
		description string
		events      []VASTEvent
		expectErr   bool
	}{
		{
			description: "No Vast Events",
			events:      nil,
			expectErr:   false,
		},
		{

			description: "Invalid Event Object",
			events: []VASTEvent{
				{
					CreateElement: "impression",
				},
				{},
			},
			expectErr: true,
		},
	}
	for _, test := range testCases {
		err := validateVASTEvents(test.events)
		assert.Equal(t, !test.expectErr, err == nil, test.description)
	}
}
