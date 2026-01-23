package model

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// VAST represents the root VAST element
type VAST struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr"`
	Ad      []*Ad    `xml:"Ad,omitempty"`
}

// Ad represents a VAST Ad element
type Ad struct {
	ID       string    `xml:"id,attr,omitempty"`
	Sequence int       `xml:"sequence,attr,omitempty"`
	InLine   *InLine   `xml:"InLine,omitempty"`
	Wrapper  *Wrapper  `xml:"Wrapper,omitempty"`
	AdSystem *AdSystem `xml:"AdSystem,omitempty"` // For backwards compatibility
}

// InLine represents a VAST InLine element
type InLine struct {
	AdSystem    *AdSystem    `xml:"AdSystem,omitempty"`
	AdTitle     string       `xml:"AdTitle,omitempty"`
	Impression  []Impression `xml:"Impression,omitempty"`
	Creatives   *Creatives   `xml:"Creatives,omitempty"`
	Extensions  *Extensions  `xml:"Extensions,omitempty"`
	Description string       `xml:"Description,omitempty"`
	Advertiser  string       `xml:"Advertiser,omitempty"`
	Pricing     *Pricing     `xml:"Pricing,omitempty"`
	Error       []Tracking   `xml:"Error,omitempty"`
	Category    []Category   `xml:"Category,omitempty"`
}

// Wrapper represents a VAST Wrapper element
type Wrapper struct {
	AdSystem       *AdSystem    `xml:"AdSystem,omitempty"`
	VASTAdTagURI   string       `xml:"VASTAdTagURI,omitempty"`
	Impression     []Impression `xml:"Impression,omitempty"`
	Creatives      *Creatives   `xml:"Creatives,omitempty"`
	Extensions     *Extensions  `xml:"Extensions,omitempty"`
	Error          []Tracking   `xml:"Error,omitempty"`
	FollowAdditionalWrappers bool `xml:"followAdditionalWrappers,attr,omitempty"`
}

// AdSystem represents the ad system information
type AdSystem struct {
	Version string `xml:"version,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// Impression represents an impression tracking URL
type Impression struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",chardata"`
}

// Tracking represents a tracking URL element
type Tracking struct {
	Event  string `xml:"event,attr,omitempty"`
	Offset string `xml:"offset,attr,omitempty"`
	Value  string `xml:",chardata"`
}

// Creatives represents the Creatives container
type Creatives struct {
	Creative []*Creative `xml:"Creative,omitempty"`
}

// Creative represents a single Creative element
type Creative struct {
	ID               string            `xml:"id,attr,omitempty"`
	Sequence         int               `xml:"sequence,attr,omitempty"`
	AdID             string            `xml:"adId,attr,omitempty"`
	Linear           *Linear           `xml:"Linear,omitempty"`
	CompanionAds     *CompanionAds     `xml:"CompanionAds,omitempty"`
	NonLinearAds     *NonLinearAds     `xml:"NonLinearAds,omitempty"`
	UniversalAdId    *UniversalAdId    `xml:"UniversalAdId,omitempty"`
	CreativeExtensions *CreativeExtensions `xml:"CreativeExtensions,omitempty"`
}

// Linear represents a Linear creative element
type Linear struct {
	Duration        string            `xml:"Duration,omitempty"`
	TrackingEvents  *TrackingEvents   `xml:"TrackingEvents,omitempty"`
	VideoClicks     *VideoClicks      `xml:"VideoClicks,omitempty"`
	MediaFiles      *MediaFiles       `xml:"MediaFiles,omitempty"`
	Icons           *Icons            `xml:"Icons,omitempty"`
	AdParameters    *AdParameters     `xml:"AdParameters,omitempty"`
	InnerXML        string            `xml:",innerxml"` // Preserve unknown elements
}

// TrackingEvents represents the TrackingEvents container
type TrackingEvents struct {
	Tracking []Tracking `xml:"Tracking,omitempty"`
}

// VideoClicks represents the VideoClicks container
type VideoClicks struct {
	ClickThrough  []ClickThrough  `xml:"ClickThrough,omitempty"`
	ClickTracking []ClickTracking `xml:"ClickTracking,omitempty"`
	CustomClick   []CustomClick   `xml:"CustomClick,omitempty"`
}

// ClickThrough represents a click-through URL
type ClickThrough struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",chardata"`
}

// ClickTracking represents a click tracking URL
type ClickTracking struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",chardata"`
}

// CustomClick represents a custom click URL
type CustomClick struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",chardata"`
}

// MediaFiles represents the MediaFiles container
type MediaFiles struct {
	MediaFile []MediaFile `xml:"MediaFile,omitempty"`
}

// MediaFile represents a single MediaFile element
type MediaFile struct {
	ID               string `xml:"id,attr,omitempty"`
	Delivery         string `xml:"delivery,attr,omitempty"`
	Type             string `xml:"type,attr,omitempty"`
	Width            int    `xml:"width,attr,omitempty"`
	Height           int    `xml:"height,attr,omitempty"`
	Codec            string `xml:"codec,attr,omitempty"`
	Bitrate          int    `xml:"bitrate,attr,omitempty"`
	MinBitrate       int    `xml:"minBitrate,attr,omitempty"`
	MaxBitrate       int    `xml:"maxBitrate,attr,omitempty"`
	Scalable         bool   `xml:"scalable,attr,omitempty"`
	MaintainAspectRatio bool `xml:"maintainAspectRatio,attr,omitempty"`
	APIFramework     string `xml:"apiFramework,attr,omitempty"`
	Value            string `xml:",chardata"`
}

// Icons represents the Icons container
type Icons struct {
	Icon []Icon `xml:"Icon,omitempty"`
}

// Icon represents a single Icon element
type Icon struct {
	Program       string         `xml:"program,attr,omitempty"`
	Width         int            `xml:"width,attr,omitempty"`
	Height        int            `xml:"height,attr,omitempty"`
	XPosition     string         `xml:"xPosition,attr,omitempty"`
	YPosition     string         `xml:"yPosition,attr,omitempty"`
	Duration      string         `xml:"duration,attr,omitempty"`
	Offset        string         `xml:"offset,attr,omitempty"`
	APIFramework  string         `xml:"apiFramework,attr,omitempty"`
	StaticResource *StaticResource `xml:"StaticResource,omitempty"`
	IconClicks    *IconClicks    `xml:"IconClicks,omitempty"`
	IconViewTracking []Tracking  `xml:"IconViewTracking,omitempty"`
}

// StaticResource represents a static resource element
type StaticResource struct {
	CreativeType string `xml:"creativeType,attr,omitempty"`
	Value        string `xml:",chardata"`
}

// IconClicks represents icon click tracking
type IconClicks struct {
	IconClickThrough  string `xml:"IconClickThrough,omitempty"`
	IconClickTracking []string `xml:"IconClickTracking,omitempty"`
}

// AdParameters represents ad parameters
type AdParameters struct {
	XMLEncoded bool   `xml:"xmlEncoded,attr,omitempty"`
	Value      string `xml:",chardata"`
}

// CompanionAds represents the CompanionAds container
type CompanionAds struct {
	Companion []Companion `xml:"Companion,omitempty"`
}

// Companion represents a single Companion ad
type Companion struct {
	ID               string            `xml:"id,attr,omitempty"`
	Width            int               `xml:"width,attr,omitempty"`
	Height           int               `xml:"height,attr,omitempty"`
	StaticResource   *StaticResource   `xml:"StaticResource,omitempty"`
	HTMLResource     string            `xml:"HTMLResource,omitempty"`
	IFrameResource   string            `xml:"IFrameResource,omitempty"`
	TrackingEvents   *TrackingEvents   `xml:"TrackingEvents,omitempty"`
	CompanionClickThrough string       `xml:"CompanionClickThrough,omitempty"`
}

// NonLinearAds represents the NonLinearAds container
type NonLinearAds struct {
	NonLinear []NonLinear `xml:"NonLinear,omitempty"`
}

// NonLinear represents a single NonLinear ad
type NonLinear struct {
	ID             string          `xml:"id,attr,omitempty"`
	Width          int             `xml:"width,attr,omitempty"`
	Height         int             `xml:"height,attr,omitempty"`
	StaticResource *StaticResource `xml:"StaticResource,omitempty"`
	NonLinearClickThrough string   `xml:"NonLinearClickThrough,omitempty"`
}

// UniversalAdId represents the universal ad identifier
type UniversalAdId struct {
	IDRegistry string `xml:"idRegistry,attr,omitempty"`
	IDValue    string `xml:"idValue,attr,omitempty"`
	Value      string `xml:",chardata"`
}

// CreativeExtensions represents creative extensions
type CreativeExtensions struct {
	CreativeExtension []Extension `xml:"CreativeExtension,omitempty"`
}

// Extensions represents the Extensions container
type Extensions struct {
	Extension []Extension `xml:"Extension,omitempty"`
}

// Extension represents a single Extension element with flexible content
type Extension struct {
	Type     string `xml:"type,attr,omitempty"`
	InnerXML string `xml:",innerxml"` // Preserve inner content
}

// Pricing represents pricing information
type Pricing struct {
	Model    string `xml:"model,attr,omitempty"`
	Currency string `xml:"currency,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// Category represents an ad category
type Category struct {
	Authority string `xml:"authority,attr,omitempty"`
	Value     string `xml:",chardata"`
}

// Parse parses VAST XML from a byte slice
func Parse(data []byte) (*VAST, error) {
	var vast VAST
	err := xml.Unmarshal(data, &vast)
	if err != nil {
		return nil, err
	}
	return &vast, nil
}

// ParseString parses VAST XML from a string
func ParseString(data string) (*VAST, error) {
	return Parse([]byte(data))
}

// Marshal serializes VAST to XML with proper formatting
func (v *VAST) Marshal() ([]byte, error) {
	data, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

// MarshalString serializes VAST to XML string
func (v *VAST) MarshalString() (string, error) {
	data, err := v.Marshal()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewEmptyVAST creates a minimal no-ad VAST response
func NewEmptyVAST(version string) *VAST {
	if version == "" {
		version = "4.0"
	}
	return &VAST{
		Version: version,
		Ad:      []*Ad{},
	}
}

// FormatDuration formats a duration in seconds to HH:MM:SS format
func FormatDuration(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// ParseDuration parses HH:MM:SS format to seconds
func ParseDuration(duration string) (int, error) {
	parts := strings.Split(duration, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}
	
	var hours, minutes, seconds int
	_, err := fmt.Sscanf(duration, "%d:%d:%d", &hours, &minutes, &seconds)
	if err != nil {
		return 0, err
	}
	
	return hours*3600 + minutes*60 + seconds, nil
}

// Helper functions for common VAST operations

// AddAd adds an Ad to the VAST response
func (v *VAST) AddAd(ad *Ad) {
	if v.Ad == nil {
		v.Ad = []*Ad{}
	}
	v.Ad = append(v.Ad, ad)
}

// GetFirstAd returns the first Ad in the VAST response
func (v *VAST) GetFirstAd() *Ad {
	if len(v.Ad) > 0 {
		return v.Ad[0]
	}
	return nil
}

// IsEmpty returns true if the VAST has no ads
func (v *VAST) IsEmpty() bool {
	return len(v.Ad) == 0
}

// NormalizeWhitespace normalizes whitespace in XML for comparison
func NormalizeWhitespace(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}

// Helper to format time for VAST
func formatTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z")
}
