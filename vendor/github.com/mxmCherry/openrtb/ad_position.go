package openrtb

// 5.4 Ad Position
//
// Position of the ad as a relative measure of visibility or prominence.
// This OpenRTB list has values derived from the Inventory Quality Guidelines (IQG).
// Practitioners should keep in sync with updates to the IQG values as published on IAB.com.
// Values “4” - “7” apply to apps per the mobile addendum to IQG version 2.1.
type AdPosition int8

const (
	AdPositionUnknown                       AdPosition = 0 // Unknown
	AdPositionAboveTheFold                  AdPosition = 1 // Above the Fold
	AdPositionMayOrMayNotBeInitiallyVisible AdPosition = 2 // DEPRECATED - May or may not be initially visible depending on screen size/resolution.
	AdPositionBelowTheFold                  AdPosition = 3 // Below the Fold
	AdPositionHeader                        AdPosition = 4 // Header
	AdPositionFooter                        AdPosition = 5 // Footer
	AdPositionSidebar                       AdPosition = 6 // Sidebar
	AdPositionFullScreen                    AdPosition = 7 // Full Screen
)

// Ptr returns pointer to own value.
func (p AdPosition) Ptr() *AdPosition {
	return &p
}

// Val safely dereferences pointer, returning default value (AdPositionUnknown) for nil.
func (p *AdPosition) Val() AdPosition {
	if p == nil {
		return AdPositionUnknown
	}
	return *p
}
