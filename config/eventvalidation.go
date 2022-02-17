package config

import (
	"errors"
	"fmt"
	"strings"

	validator "github.com/asaskevich/govalidator"
	"github.com/golang/glog"
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

// validate verifies the events object  and returns error if at least one is invalid.
func (e Events) validate(errs []error) []error {
	if e.Enabled { // validate only if events are enabled
		glog.Warning(`Don't enable this feature. It is still under developmment - https://github.com/prebid/prebid-server/issues/1725`)
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
	if events != nil {
		for i, event := range events {
			if err := validateVASTEvent(event, i); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateVASTEvent validates event object and  returns error if at least one is invalid
func validateVASTEvent(event VASTEvent, index int) error {
	if !isValidCreateElement(event.CreateElement) {
		return fmt.Errorf("Invalid events.vast_events[%d].create_element", index)
	}

	// VASTEvent.ExcludeDefaultURL assumed to be false by default
	if !isValidType(event) {
		if isTrackingEvent(event) {
			var ele []string
			for k := range vastEventElementMap {
				ele = append(ele, string(k))
			}
			return fmt.Errorf("Missing or Invalid events.vast_events[%d].type. Valid values are %v", index, strings.Join(ele, ", "))
		}
		return fmt.Errorf("events.vast_events[%d].type is not applicable for create element '%s'", index, event.CreateElement)
	}
	for i, url := range event.URLs {
		if !isValidURL(url) {
			return fmt.Errorf("Invalid events.vast_events[%d].urls[%d]", index, i)
		}
	}
	// ensure at least one valid url exists when default URL to be excluded
	if event.ExcludeDefaultURL && len(event.URLs) == 0 {
		return fmt.Errorf("Please provide at least one valid URL in events.vast_events[%d].urls or set events.vast_events[%d].exclude_default_url=false", index, index)
	}

	return nil // no errors
}

// isValidCreateElement checks create_element has valid value
// if value is value returns true, otherwise false
func isValidCreateElement(element VASTEventElement) bool {
	// validate create element
	if _, ok := vastEventElementMap[element]; ok {
		return true
	}
	return false
}

// isValidtype checks if valid type is provided in case event is of type tracking.
// in case of other events this value must be empty
func isValidType(event VASTEvent) bool {
	if isTrackingEvent(event) {
		if _, ok := trackingEventTypeMap[event.Type]; ok {
			// valid event type for create element tracking
			return true
		}
		return false
	}
	return len(event.Type) == 0 // event.type must be empty in case create element is not tracking
}

// isValidURL validates the event URL
func isValidURL(eventURL string) bool {
	return validator.IsURL(eventURL) && validator.IsRequestURL(eventURL)
}

// isTrackingEvent returns true if event object contains event.CreateElement == "tracking"
func isTrackingEvent(event VASTEvent) bool {
	return event.CreateElement == TrackingVASTElement
}
