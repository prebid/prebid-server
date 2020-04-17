package openrtb

// 5.5 Expandable Direction
//
// Directions in which an expandable ad may expand, given the positioning of the ad unit on the page and constraints imposed by the content.
type ExpandableDirection int8

const (
	ExpandableDirectionLeft       ExpandableDirection = 1 // Left
	ExpandableDirectionRight      ExpandableDirection = 2 // Right
	ExpandableDirectionUp         ExpandableDirection = 3 // Up
	ExpandableDirectionDown       ExpandableDirection = 4 // Down
	ExpandableDirectionFullScreen ExpandableDirection = 5 // Full Screen
)
