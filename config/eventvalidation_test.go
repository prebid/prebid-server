package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// isValidCreateElement tests
func TestIsValidCreateElementEmpty(t *testing.T) {
	assert.False(t, isValidCreateElement(""))
}
func TestIsValidCreateElementFalse(t *testing.T) {
	assert.False(t, isValidCreateElement("invalid_element"))
}
func TestIsValidCreateElementTrue(t *testing.T) {
	assert.True(t, isValidCreateElement("tracking"))
}
func TestIsValidCreateElementCaseSensitive(t *testing.T) {
	assert.False(t, isValidCreateElement("Tracking"))
}

// isValidType tests
func TestIsValidTypeEmpty(t *testing.T) {
	assert.True(t, isValidType(VASTEvent{
		Type: "",
	}))
}

// type shuold NOT be empty when create element is tracking
func TestIsValidTypeEmptyWithCreateElementTracking(t *testing.T) {
	assert.False(t, isValidType(VASTEvent{
		CreateElement: "tracking",
		Type:          "",
	}))
}

func TestIsValidTypeFalse(t *testing.T) {
	assert.False(t, isValidType(VASTEvent{
		CreateElement: "tracking",
		Type:          "invalid_type",
	}))
}
func TestIsValidTypeTrue(t *testing.T) {
	assert.True(t, isValidType(VASTEvent{
		CreateElement: "tracking",
		Type:          "complete",
	}))
}

func TestIsValidTypeCaseSensitive(t *testing.T) {
	assert.False(t, isValidType(VASTEvent{
		CreateElement: "tracking",
		Type:          "COMPlete",
	}))
}

// isValidURL tests
func TestIsValidURLEmpty(t *testing.T) {
	assert.False(t, isValidURL(""))
}
func TestIsValidURLTrue(t *testing.T) {
	assert.True(t, isValidURL("http://PREBID.ORG"))
	assert.True(t, isValidURL("https://PBS_HOST/event?t=##PBS-EVENTTYPE##&vtype=##PBS-VASTEVENT##&b=##PBS-BIDID##&f=i&a=##PBS-ACCOUNTID##&ts=##PBS-TIMESTAMP##&bidder=##PBS-BIDDER##&int=##PBS-INTEGRATION##&mt=##PBS-MEDIATYPE##&ch=##PBS-CHANNEL##&aid=##PBS-AUCTIONID##&l=##PBS-LINEID##"))

}
func TestIsValidURLFalse(t *testing.T) {
	assert.False(t, isValidURL("HTTP://PREBID.ORG"))
	assert.False(t, isValidURL("HTTPE://PREBID.ORG"))
	assert.False(t, isValidURL("HTTPE://PREBID.ORG:8000"))
	assert.False(t, isValidURL("https://prebid.org::8000"))
}
func TestIsValidURLWithPortTrue(t *testing.T) {
	assert.True(t, isValidURL("https://prebid.org:8000"))
}

// isTrackingEvent tests
func TestIsTrackingEventEmpty(t *testing.T) {
	assert.False(t, isTrackingEvent(VASTEvent{}))
}
func TestIsTrackingEventTrue(t *testing.T) {
	assert.True(t, isTrackingEvent(VASTEvent{
		CreateElement: "tracking",
	}))
}
func TestIsTrackingEventCaseSensitive(t *testing.T) {
	assert.False(t, isTrackingEvent(VASTEvent{
		CreateElement: "Tracking",
	}))
}

// validateVASTEvent tests

// tests for
// ** there must be at least one url in each vast event object

// expect no error as ExcludeDefaultURL=false will consider
// host level default_url
func TestVASTEventWithNoURLAndExcludeDefaultURLFalse(t *testing.T) {
	assert.Nil(t, validateVASTEvent(VASTEvent{
		CreateElement:     "tracking",
		Type:              "start",
		ExcludeDefaultURL: false,
		URLs:              nil,
	}, 0))
}

// expect error (non nil object) as ExcludeDefaultURL=true and no urls
func TestVASTEventWithNoURLAndExcludeDefaultURLTrue(t *testing.T) {
	assert.NotNil(t, validateVASTEvent(VASTEvent{
		CreateElement:     "tracking",
		Type:              "start",
		ExcludeDefaultURL: true,
		URLs:              nil,
	}, 0))
}

// expect error  i.e. non nil object when create element is invalid
func TestValidateVASTEventInvalidCreateElement(t *testing.T) {
	assert.NotNil(t, validateVASTEvent(VASTEvent{}, 0))
}

// expect error (non nil object) when create_element=tracking and type is invalid
func TestValidateVASTEventInvalidTypeForTrackingCreateElement(t *testing.T) {
	assert.NotNil(t, validateVASTEvent(VASTEvent{
		CreateElement: "tracking",
		Type:          "some_type",
	}, 0))
}

// expect error (non nil object) when URLs[] contains invalid values
func TestValidateVASTEventInvalidURLs(t *testing.T) {
	assert.NotNil(t, validateVASTEvent(VASTEvent{
		CreateElement: "impression",
		URLs:          []string{"http://url.com?k1=v1&k2=###PBS-MACRO##", "httpE://invalid.url"},
	}, 0))
}

// expect error (non nil object) when type is valid but create-element is not 'tracking'
func TestValidateVASTEventTypeNotApplicable(t *testing.T) {
	assert.NotNil(t, validateVASTEvent(VASTEvent{
		CreateElement: "impression",
		Type:          "some_type",
	}, 0))
}

// Validate tests

// Expect error (non nil object) becase default_url = "" i.e. invalid
func TestValidateEventsEnabled(t *testing.T) {
	e := Events{Enabled: true}
	assert.NotNil(t, e.validate(make([]error, 0)))
}

// Expect no error when no vast events are there but
// enabled = true and valid default_url is present
func TestValidateEventsEmptyVASTEvents(t *testing.T) {
	e := Events{Enabled: true, DefaultURL: "http://prebid.org"}
	assert.Empty(t, e.validate(make([]error, 0)))
}

// Expect error (non nil object) becase of invalid vast events
func TestValidateInvalidVASTEvents(t *testing.T) {
	e := Events{Enabled: true, DefaultURL: "http://prebid.org",
		VASTEvents: []VASTEvent{
			{}, // this is invalid vast event as no create element specified
		}}
	assert.NotNil(t, e.validate(make([]error, 0)))
}

// validateVASTEvents tests

// Expect no error if vast events array is empty
func TestValidateVASTEventsNilArray(t *testing.T) {
	assert.Nil(t, validateVASTEvents(nil))
}

// Expect error (non nil object) when vast events array contains
// invalid event object
func TestValidateVASTEventsWithInvalidEvent(t *testing.T) {
	events := []VASTEvent{
		{
			CreateElement: "impression",
		},
		{}, // this is invalid object as no create element present
	}
	assert.NotNil(t, validateVASTEvents(events))
}
