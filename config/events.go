package config

import (
	"errors"
	"fmt"
	"strings"

	validator "github.com/asaskevich/govalidator"
)

// VASTEventElement indicates valid VAST event element
type VASTEventElement string

const (
	ImpressionVASTElement             VASTEventElement = "impression"
	TrackingVASTElement               VASTEventElement = "tracking"
	ClickTrackingVASTElement          VASTEventElement = "clicktracking"
	CompanionClickThroughVASTElement  VASTEventElement = "companionclickthrough"
	ErrorVASTElement                  VASTEventElement = "error"
	NonLinearClickTrackingVASTElement VASTEventElement = "nonlinearclicktracking"
)

var vastEventElementMap = map[VASTEventElement]struct{}{
	ImpressionVASTElement:             {},
	TrackingVASTElement:               {},
	ClickTrackingVASTElement:          {},
	CompanionClickThroughVASTElement:  {},
	ErrorVASTElement:                  {},
	NonLinearClickTrackingVASTElement: {},
}

// TrackingEventType indicates quartile events
type TrackingEventType string

const (
	Start         TrackingEventType = "start"
	FirstQuartile TrackingEventType = "firstQuartile"
	MidPoint      TrackingEventType = "midPoint"
	ThirdQuartile TrackingEventType = "thirdQuartile"
	Complete      TrackingEventType = "complete"
)

var trackingEventTypeMap = map[TrackingEventType]struct{}{
	Start:         {},
	FirstQuartile: {},
	MidPoint:      {},
	ThirdQuartile: {},
	Complete:      {},
}

// VASTEvent indicates the configurations required for injecting VAST event trackers within
// VAST XML
type VASTEvent struct {
	CreateElement     VASTEventElement  `mapstructure:"create_element" json:"create_element"`
	Type              TrackingEventType `mapstructure:"type" json:"type"`
	ExcludeDefaultURL bool              `mapstructure:"exclude_default_url" json:"exclude_default_url"`
	URLs              []string          `mapstructure:"urls" json:"urls"`
}

// Events indicates the various types of events to be captured typically for injecting tracker URLs
// within the VAST XML
// Don't enable this feature. It is still under developmment. Please follow https://github.com/prebid/prebid-server/issues/1725 for more updates
type Events struct {
	Enabled    bool        `mapstructure:"enabled" json:"enabled"`
	DefaultURL string      `mapstructure:"default_url" json:"default_url"`
	VASTEvents []VASTEvent `mapstructure:"vast_events" json:"vast_events,omitempty"`
}

// validate verifies the events object  and returns error if at least one is invalid.
func (e Events) validate(errs []error) []error {
	if e.Enabled {
		if !isValidURL(e.DefaultURL) {
			return append(errs, errors.New("Invalid events.default_url"))
		}
		err := validateVASTEvents(e.VASTEvents)
		if err != nil {
			return append(errs, err)
		}
	}
	return errs
}

// validateVASTEvents verifies the all VASTEvent objects and returns error if at least one is invalid.
func validateVASTEvents(events []VASTEvent) error {
	for i, event := range events {
		if err := event.validate(); err != nil {
			return fmt.Errorf(err.Error(), i, i)
		}
	}
	return nil
}

// validate validates event object and  returns error if at least one is invalid
func (e VASTEvent) validate() error {
	if !e.CreateElement.isValid() {
		return fmt.Errorf("Invalid events.vast_events[%s].create_element", "%d")
	}
	validType := e.Type.isValid()

	if e.isTrackingEvent() && !validType {
		var ele []string
		for k := range vastEventElementMap {
			ele = append(ele, string(k))
		}
		return fmt.Errorf("Missing or Invalid events.vast_events[%s].type. Valid values are %v", "%d", strings.Join(ele, ", "))
	}
	if validType && !e.isTrackingEvent() {
		return fmt.Errorf("events.vast_events[%s].type is not applicable for create element '%s'", "%d", e.CreateElement)
	}
	for i, url := range e.URLs {
		if !isValidURL(url) {
			return fmt.Errorf("Invalid events.vast_events[%s].urls[%d]", "%d", i)
		}
	}
	// ensure at least one valid url exists when default URL to be excluded
	if e.ExcludeDefaultURL && len(e.URLs) == 0 {
		return fmt.Errorf("Please provide at least one valid URL in events.vast_events[%s].urls or set events.vast_events[%s].exclude_default_url=false", "%d", "%d")
	}

	return nil // no errors
}

// isValid checks create_element has valid value
// if value is value returns true, otherwise false
func (element VASTEventElement) isValid() bool {
	// validate create element
	if _, ok := vastEventElementMap[element]; ok {
		return true
	}
	return false
}

// isValid checks if valid type is provided (case-sensitive)
func (t TrackingEventType) isValid() bool {
	_, valid := trackingEventTypeMap[t]
	return valid
}

// isValidURL validates the event URL
func isValidURL(eventURL string) bool {
	return validator.IsURL(eventURL) && validator.IsRequestURL(eventURL)
}

// isTrackingEvent returns true if event object contains event.CreateElement == "tracking"
func (e VASTEvent) isTrackingEvent() bool {
	return e.CreateElement == TrackingVASTElement
}
