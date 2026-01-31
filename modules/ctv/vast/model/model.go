package model

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// Vast represents the root VAST document
type Vast struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr,omitempty"`
	Ads     []Ad     `xml:"Ad"`
}

// Ad represents a single ad in a VAST response
type Ad struct {
	ID       string  `xml:"id,attr,omitempty"`
	Sequence int     `xml:"sequence,attr,omitempty"`
	InLine   *InLine `xml:"InLine,omitempty"`
	InnerXML string  `xml:",innerxml"`
}

// InLine represents an inline VAST ad
type InLine struct {
	AdSystem   string      `xml:"AdSystem,omitempty"`
	AdTitle    string      `xml:"AdTitle,omitempty"`
	Advertiser string      `xml:"Advertiser,omitempty"`
	Pricing    *Pricing    `xml:"Pricing,omitempty"`
	Creatives  *Creatives  `xml:"Creatives,omitempty"`
	Extensions *Extensions `xml:"Extensions,omitempty"`
	InnerXML   string      `xml:",innerxml"`
}

// Pricing contains pricing information (VAST 3.0+)
type Pricing struct {
	Model    string `xml:"model,attr,omitempty"`
	Currency string `xml:"currency,attr,omitempty"`
	Value    string `xml:",chardata"`
}

// Creatives contains the creative assets
type Creatives struct {
	Creatives []Creative `xml:"Creative"`
}

// Creative represents a VAST creative
type Creative struct {
	ID       string  `xml:"id,attr,omitempty"`
	Sequence int     `xml:"sequence,attr,omitempty"`
	Linear   *Linear `xml:"Linear,omitempty"`
}

// Linear represents a linear video creative
type Linear struct {
	Duration   string       `xml:"Duration,omitempty"`
	MediaFiles *MediaFiles  `xml:"MediaFiles,omitempty"`
	InnerXML   string       `xml:",innerxml"`
}

// MediaFiles contains video files
type MediaFiles struct {
	MediaFiles []MediaFile `xml:"MediaFile"`
}

// MediaFile represents a video file
type MediaFile struct {
	Delivery string `xml:"delivery,attr,omitempty"`
	Type     string `xml:"type,attr,omitempty"`
	Width    int    `xml:"width,attr,omitempty"`
	Height   int    `xml:"height,attr,omitempty"`
	URI      string `xml:",chardata"`
}

// Extensions contains VAST extensions
type Extensions struct {
	Extensions []Extension `xml:"Extension"`
}

// Extension represents a single extension element
type Extension struct {
	Type     string `xml:"type,attr,omitempty"`
	InnerXML string `xml:",innerxml"`
}

// SecToHHMMSS converts seconds to HH:MM:SS format
func SecToHHMMSS(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// BuildNoAdVast creates an empty VAST response (no ads)
func BuildNoAdVast(version string) []byte {
	if version == "" {
		version = "3.0"
	}
	
	vast := Vast{
		Version: version,
		Ads:     []Ad{},
	}
	
	xmlBytes, err := xml.MarshalIndent(vast, "", "  ")
	if err != nil {
		// Fallback to simple string if marshal fails
		return []byte(fmt.Sprintf(`<VAST version="%s"></VAST>`, version))
	}
	
	// Prepend XML header
	return append([]byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"), xmlBytes...)
}

// BuildSkeletonInlineVast creates a skeleton VAST with one ad placeholder
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
					AdSystem: "Prebid",
					AdTitle:  "Ad",
					Creatives: &Creatives{
						Creatives: []Creative{
							{
								ID: "1",
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

// VastAd is an alias for Ad to match the interface expectations
type VastAd = Ad

// ParseVastAdm parses a VAST XML string into a Vast struct.
// Returns error if the string doesn't contain "<VAST" or if XML parsing fails.
// Preserves unknown elements via InnerXML fields. Empty Version attribute is allowed.
func ParseVastAdm(adm string) (*Vast, error) {
	if !strings.Contains(adm, "<VAST") {
		return nil, errors.New("adm does not contain <VAST tag")
	}

	var vast Vast
	err := xml.Unmarshal([]byte(adm), &vast)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VAST XML: %w", err)
	}

	// Version can be empty - we don't enforce it
	return &vast, nil
}

// ReceiverConfigForParser is a minimal config interface for parsing fallback behavior.
// This avoids circular imports between model and parent package.
type ReceiverConfigForParser interface {
	GetAllowSkeletonVast() bool
	GetVastVersionDefault() string
}

// ParseVastOrSkeleton attempts to parse VAST XML, with fallback to skeleton on failure.
// Returns:
//   - *Vast: parsed VAST or skeleton if parsing fails and AllowSkeletonVast is true
//   - []string: warnings (e.g., "failed to parse VAST, using skeleton")
//   - error: only if parsing fails and AllowSkeletonVast is false
func ParseVastOrSkeleton(adm string, cfg ReceiverConfigForParser) (*Vast, []string, error) {
	vast, err := ParseVastAdm(adm)
	if err == nil {
		return vast, nil, nil
	}

	// Parsing failed - check if skeleton fallback is allowed
	if cfg.GetAllowSkeletonVast() {
		version := cfg.GetVastVersionDefault()
		skeleton := BuildSkeletonInlineVast(version)
		warnings := []string{fmt.Sprintf("failed to parse VAST: %v, using skeleton", err)}
		return skeleton, warnings, nil
	}

	// No fallback allowed - return the error
	return nil, nil, fmt.Errorf("failed to parse VAST and skeleton fallback disabled: %w", err)
}

