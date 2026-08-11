package model

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Vast represents the root VAST XML element.
type Vast struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr,omitempty"`
	Ads     []Ad     `xml:"Ad"`
}

// Ad represents a VAST Ad element.
type Ad struct {
	ID       string   `xml:"id,attr,omitempty"`
	Sequence int      `xml:"sequence,attr,omitempty"`
	InLine   *InLine  `xml:"InLine,omitempty"`
	Wrapper  *Wrapper `xml:"Wrapper,omitempty"`
	// InnerXML preserves unknown nodes if needed
	InnerXML string `xml:",innerxml"`
}

// InLine represents a VAST InLine element containing the ad data.
type InLine struct {
	AdSystem    *AdSystem    `xml:"AdSystem,omitempty"`
	AdTitle     string       `xml:"AdTitle,omitempty"`
	Advertiser  string       `xml:"Advertiser,omitempty"`
	Description string       `xml:"Description,omitempty"`
	Error       string       `xml:"Error,omitempty"`
	Impressions []Impression `xml:"Impression,omitempty"`
	Pricing     *Pricing     `xml:"Pricing,omitempty"`
	Creatives   *Creatives   `xml:"Creatives,omitempty"`
	Extensions  *Extensions  `xml:"Extensions,omitempty"`
	// InnerXML preserves unknown nodes
	InnerXML string `xml:",innerxml"`
}

// Wrapper represents a VAST Wrapper element for wrapped ads.
type Wrapper struct {
	AdSystem     *AdSystem    `xml:"AdSystem,omitempty"`
	VASTAdTagURI string       `xml:"VASTAdTagURI,omitempty"`
	Error        string       `xml:"Error,omitempty"`
	Impressions  []Impression `xml:"Impression,omitempty"`
	Creatives    *Creatives   `xml:"Creatives,omitempty"`
	Extensions   *Extensions  `xml:"Extensions,omitempty"`
	// InnerXML preserves unknown nodes
	InnerXML string `xml:",innerxml"`
}

// AdSystem identifies the ad server that returned the ad.
type AdSystem struct {
	Version string `xml:"version,attr,omitempty"`
	Value   string `xml:",chardata"`
}

// Impression represents an impression tracking URL.
type Impression struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// Pricing contains pricing information for the ad.
type Pricing struct {
	Model    string `xml:"model,attr,omitempty"`
	Currency string `xml:"currency,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// Creatives contains a list of Creative elements.
type Creatives struct {
	Creative []Creative `xml:"Creative,omitempty"`
}

// Creative represents a VAST Creative element.
type Creative struct {
	ID            string         `xml:"id,attr,omitempty"`
	AdID          string         `xml:"adId,attr,omitempty"`
	Sequence      int            `xml:"sequence,attr,omitempty"`
	UniversalAdID *UniversalAdId `xml:"UniversalAdId,omitempty"`
	Linear        *Linear        `xml:"Linear,omitempty"`
	// InnerXML preserves unknown nodes
	InnerXML string `xml:",innerxml"`
}

// UniversalAdId provides a unique creative identifier across systems.
type UniversalAdId struct {
	IDRegistry string `xml:"idRegistry,attr,omitempty"`
	IDValue    string `xml:"idValue,attr,omitempty"`
	Value      string `xml:",chardata"`
}

// Linear represents a linear (video) creative.
type Linear struct {
	SkipOffset     string          `xml:"skipoffset,attr,omitempty"`
	Duration       string          `xml:"Duration,omitempty"`
	MediaFiles     *MediaFiles     `xml:"MediaFiles,omitempty"`
	VideoClicks    *VideoClicks    `xml:"VideoClicks,omitempty"`
	TrackingEvents *TrackingEvents `xml:"TrackingEvents,omitempty"`
	AdParameters   *AdParameters   `xml:"AdParameters,omitempty"`
	// InnerXML preserves unknown nodes
	InnerXML string `xml:",innerxml"`
}

// MediaFiles contains a list of MediaFile elements.
type MediaFiles struct {
	MediaFile []MediaFile `xml:"MediaFile,omitempty"`
}

// MediaFile represents a video media file.
type MediaFile struct {
	ID                  string `xml:"id,attr,omitempty"`
	Delivery            string `xml:"delivery,attr,omitempty"`
	Type                string `xml:"type,attr,omitempty"`
	Width               int    `xml:"width,attr,omitempty"`
	Height              int    `xml:"height,attr,omitempty"`
	Bitrate             int    `xml:"bitrate,attr,omitempty"`
	MinBitrate          int    `xml:"minBitrate,attr,omitempty"`
	MaxBitrate          int    `xml:"maxBitrate,attr,omitempty"`
	Scalable            bool   `xml:"scalable,attr,omitempty"`
	MaintainAspectRatio bool   `xml:"maintainAspectRatio,attr,omitempty"`
	Codec               string `xml:"codec,attr,omitempty"`
	Value               string `xml:",cdata"`
}

// VideoClicks contains click tracking URLs for video ads.
type VideoClicks struct {
	ClickThrough  *ClickThrough   `xml:"ClickThrough,omitempty"`
	ClickTracking []ClickTracking `xml:"ClickTracking,omitempty"`
	CustomClick   []CustomClick   `xml:"CustomClick,omitempty"`
}

// ClickThrough represents the landing page URL.
type ClickThrough struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// ClickTracking represents a click tracking URL.
type ClickTracking struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// CustomClick represents a custom click URL.
type CustomClick struct {
	ID    string `xml:"id,attr,omitempty"`
	Value string `xml:",cdata"`
}

// TrackingEvents contains tracking URLs for various playback events.
type TrackingEvents struct {
	Tracking []Tracking `xml:"Tracking,omitempty"`
}

// Tracking represents a single tracking event.
type Tracking struct {
	Event  string `xml:"event,attr,omitempty"`
	Offset string `xml:"offset,attr,omitempty"`
	Value  string `xml:",cdata"`
}

// AdParameters holds custom parameters for the ad.
type AdParameters struct {
	XMLEncoded bool   `xml:"xmlEncoded,attr,omitempty"`
	Value      string `xml:",cdata"`
}

// Extensions contains a list of Extension elements.
type Extensions struct {
	Extension []ExtensionXML `xml:"Extension,omitempty"`
}

// ExtensionXML represents a VAST extension element.
type ExtensionXML struct {
	Type string `xml:"type,attr,omitempty"`
	// InnerXML preserves the extension content
	InnerXML string `xml:",innerxml"`
}

// SecToHHMMSS converts seconds to HH:MM:SS format used in VAST Duration.
func SecToHHMMSS(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// FormatPrice formats a bid price with up to 4 decimal places, trimming
// trailing zeros. Used by all enrichers to ensure consistent price
// representation in VAST XML (e.g. 5.5 not 5.5000, 1.25 not 1.2500).
func FormatPrice(price float64) string {
	s := fmt.Sprintf("%.4f", price)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// BuildNoAdVast creates a VAST response indicating no ad is available.
// This is a valid VAST document with no Ad elements.
func BuildNoAdVast(version string) []byte {
	if version == "" {
		version = "3.0"
	}
	vast := Vast{
		Version: version,
		Ads:     []Ad{},
	}
	output, err := xml.MarshalIndent(vast, "", "  ")
	if err != nil {
		// Fallback to minimal valid VAST
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><VAST version="%s"/>`, version))
	}
	return append([]byte(xml.Header), output...)
}

// BuildSkeletonInlineVast creates a minimal VAST document with one InLine ad.
// This skeleton can be used as a template to fill in with actual ad data.
func BuildSkeletonInlineVast(version string) *Vast {
	if version == "" {
		version = "3.0"
	}
	return &Vast{
		Version: version,
		Ads: []Ad{
			{
				ID:       "1",
				Sequence: 1,
				InLine: &InLine{
					AdSystem: &AdSystem{
						Value: "PBS-CTV",
					},
					AdTitle: "Ad",
					Creatives: &Creatives{
						Creative: []Creative{
							{
								ID:       "1",
								Sequence: 1,
								Linear: &Linear{
									Duration: "00:00:00",
								},
							},
						},
					},
				},
			},
		},
	}
}

// BuildSkeletonInlineVastWithDuration creates a minimal VAST document with specified duration.
func BuildSkeletonInlineVastWithDuration(version string, durationSec int) *Vast {
	vast := BuildSkeletonInlineVast(version)
	if len(vast.Ads) > 0 && vast.Ads[0].InLine != nil &&
		vast.Ads[0].InLine.Creatives != nil &&
		len(vast.Ads[0].InLine.Creatives.Creative) > 0 &&
		vast.Ads[0].InLine.Creatives.Creative[0].Linear != nil {
		vast.Ads[0].InLine.Creatives.Creative[0].Linear.Duration = SecToHHMMSS(durationSec)
	}
	return vast
}

// DeepCopy returns a fully independent copy of the Ad and all its nested
// pointer fields. Modifying any field in the returned Ad (including InLine,
// Creatives, Pricing, Extensions, etc.) will not affect the original.
func (a *Ad) DeepCopy() *Ad {
	if a == nil {
		return nil
	}
	copy := *a
	copy.InLine = deepCopyInLine(a.InLine)
	copy.Wrapper = deepCopyWrapper(a.Wrapper)
	return &copy
}

// deepCopyInLine returns a deep copy of an InLine element.
func deepCopyInLine(src *InLine) *InLine {
	if src == nil {
		return nil
	}
	c := *src
	c.AdSystem = deepCopyAdSystem(src.AdSystem)
	if src.Impressions != nil {
		c.Impressions = make([]Impression, len(src.Impressions))
		copy(c.Impressions, src.Impressions)
	}
	c.Pricing = deepCopyPricing(src.Pricing)
	c.Creatives = deepCopyCreatives(src.Creatives)
	c.Extensions = deepCopyExtensions(src.Extensions)
	return &c
}

// deepCopyWrapper returns a deep copy of a Wrapper element.
func deepCopyWrapper(src *Wrapper) *Wrapper {
	if src == nil {
		return nil
	}
	c := *src
	c.AdSystem = deepCopyAdSystem(src.AdSystem)
	if src.Impressions != nil {
		c.Impressions = make([]Impression, len(src.Impressions))
		copy(c.Impressions, src.Impressions)
	}
	c.Creatives = deepCopyCreatives(src.Creatives)
	c.Extensions = deepCopyExtensions(src.Extensions)
	return &c
}

func deepCopyAdSystem(src *AdSystem) *AdSystem {
	if src == nil {
		return nil
	}
	c := *src
	return &c
}

func deepCopyPricing(src *Pricing) *Pricing {
	if src == nil {
		return nil
	}
	c := *src
	return &c
}

func deepCopyCreatives(src *Creatives) *Creatives {
	if src == nil {
		return nil
	}
	c := Creatives{}
	if src.Creative != nil {
		c.Creative = make([]Creative, len(src.Creative))
		for i, cr := range src.Creative {
			cc := cr
			if cr.UniversalAdID != nil {
				uaid := *cr.UniversalAdID
				cc.UniversalAdID = &uaid
			}
			cc.Linear = deepCopyLinear(cr.Linear)
			c.Creative[i] = cc
		}
	}
	return &c
}

func deepCopyLinear(src *Linear) *Linear {
	if src == nil {
		return nil
	}
	c := *src
	c.MediaFiles = deepCopyMediaFiles(src.MediaFiles)
	c.VideoClicks = deepCopyVideoClicks(src.VideoClicks)
	c.TrackingEvents = deepCopyTrackingEvents(src.TrackingEvents)
	if src.AdParameters != nil {
		ap := *src.AdParameters
		c.AdParameters = &ap
	}
	return &c
}

func deepCopyMediaFiles(src *MediaFiles) *MediaFiles {
	if src == nil {
		return nil
	}
	c := MediaFiles{}
	if src.MediaFile != nil {
		c.MediaFile = make([]MediaFile, len(src.MediaFile))
		copy(c.MediaFile, src.MediaFile)
	}
	return &c
}

func deepCopyVideoClicks(src *VideoClicks) *VideoClicks {
	if src == nil {
		return nil
	}
	c := VideoClicks{}
	if src.ClickThrough != nil {
		ct := *src.ClickThrough
		c.ClickThrough = &ct
	}
	if src.ClickTracking != nil {
		c.ClickTracking = make([]ClickTracking, len(src.ClickTracking))
		copy(c.ClickTracking, src.ClickTracking)
	}
	if src.CustomClick != nil {
		c.CustomClick = make([]CustomClick, len(src.CustomClick))
		copy(c.CustomClick, src.CustomClick)
	}
	return &c
}

func deepCopyTrackingEvents(src *TrackingEvents) *TrackingEvents {
	if src == nil {
		return nil
	}
	c := TrackingEvents{}
	if src.Tracking != nil {
		c.Tracking = make([]Tracking, len(src.Tracking))
		copy(c.Tracking, src.Tracking)
	}
	return &c
}

func deepCopyExtensions(src *Extensions) *Extensions {
	if src == nil {
		return nil
	}
	c := Extensions{}
	if src.Extension != nil {
		c.Extension = make([]ExtensionXML, len(src.Extension))
		copy(c.Extension, src.Extension)
	}
	return &c
}

// Marshal serializes the Vast struct to XML bytes with XML header.
func (v *Vast) Marshal() ([]byte, error) {
	// Clear InnerXML fields to prevent duplicate content
	v.clearInnerXML()
	output, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), output...), nil
}

// MarshalCompact serializes the Vast struct to XML bytes without indentation.
func (v *Vast) MarshalCompact() ([]byte, error) {
	// Clear InnerXML fields to prevent duplicate content
	v.clearInnerXML()
	output, err := xml.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), output...), nil
}

// clearInnerXML clears InnerXML only on nodes that are directly modified by enrichment.
// Nodes higher in the tree (Ad, InLine, Wrapper) get their InnerXML cleared so structured
// fields (Pricing, Advertiser, etc.) are serialized without duplication.
// Creative, Linear, MediaFiles, TrackingEvents and other leaf nodes are preserved —
// this keeps DSP-specific extensions and unknown elements intact (BUG 4 fix).
func (v *Vast) clearInnerXML() {
	for i := range v.Ads {
		v.Ads[i].InnerXML = ""
		if v.Ads[i].InLine != nil {
			v.Ads[i].InLine.InnerXML = ""
			// Creative.InnerXML and Linear.InnerXML are intentionally preserved
			// to keep MediaFiles, TrackingEvents, and DSP extensions intact.
		}
		if v.Ads[i].Wrapper != nil {
			v.Ads[i].Wrapper.InnerXML = ""
		}
	}
}

// Unmarshal parses XML bytes into a Vast struct.
func Unmarshal(data []byte) (*Vast, error) {
	var vast Vast
	if err := xml.Unmarshal(data, &vast); err != nil {
		return nil, err
	}
	return &vast, nil
}
