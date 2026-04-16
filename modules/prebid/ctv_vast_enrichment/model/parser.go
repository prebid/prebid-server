package model

import (
	"encoding/xml"
	"errors"
	"strings"
)

// ErrNotVAST indicates the input string does not appear to be VAST XML.
var ErrNotVAST = errors.New("input does not contain VAST XML")

// ErrVASTParseFailure indicates the VAST XML could not be parsed.
var ErrVASTParseFailure = errors.New("failed to parse VAST XML")

// ParseVastAdm parses a VAST XML string from an OpenRTB bid's AdM field.
// Returns an error if the input doesn't contain "<VAST" or cannot be parsed.
// Unknown elements are preserved via InnerXML fields where possible.
func ParseVastAdm(adm string) (*Vast, error) {
	// Check if input looks like VAST
	if !strings.Contains(adm, "<VAST") {
		return nil, ErrNotVAST
	}

	var vast Vast
	if err := xml.Unmarshal([]byte(adm), &vast); err != nil {
		return nil, errors.Join(ErrVASTParseFailure, err)
	}

	// Version being empty is acceptable - just means it wasn't specified
	// We don't error on missing version, caller can apply defaults

	return &vast, nil
}

// ParserConfig holds configuration for VAST parsing behavior.
// This mirrors the fields needed from ReceiverConfig for parsing operations.
type ParserConfig struct {
	// AllowSkeletonVast allows returning a skeleton VAST when parsing fails.
	AllowSkeletonVast bool
	// VastVersionDefault is the default VAST version to use for skeleton.
	VastVersionDefault string
}

// ParseVastOrSkeleton attempts to parse VAST XML and falls back to a skeleton if allowed.
// Returns:
//   - parsed *Vast on success
//   - skeleton *Vast with warning if parse fails and AllowSkeletonVast is true
//   - error if parse fails and AllowSkeletonVast is false
func ParseVastOrSkeleton(adm string, cfg ParserConfig) (*Vast, []string, error) {
	var warnings []string

	// Try to parse the VAST
	vast, err := ParseVastAdm(adm)
	if err == nil {
		return vast, warnings, nil
	}

	// Parse failed - check if we should return skeleton
	if !cfg.AllowSkeletonVast {
		return nil, warnings, err
	}

	// Return skeleton VAST with warning
	version := cfg.VastVersionDefault
	if version == "" {
		version = "3.0"
	}

	warnings = append(warnings, "VAST parse failed, using skeleton: "+err.Error())
	skeleton := BuildSkeletonInlineVast(version)
	return skeleton, warnings, nil
}

// ParseVastFromBytes parses VAST XML from a byte slice.
func ParseVastFromBytes(data []byte) (*Vast, error) {
	return ParseVastAdm(string(data))
}

// ExtractFirstAd extracts the first Ad from a parsed VAST.
// Returns nil if no ads are present.
func ExtractFirstAd(vast *Vast) *Ad {
	if vast == nil || len(vast.Ads) == 0 {
		return nil
	}
	return &vast.Ads[0]
}

// ExtractDuration attempts to extract the duration string from a parsed VAST.
// Returns empty string if duration cannot be found.
func ExtractDuration(vast *Vast) string {
	if vast == nil || len(vast.Ads) == 0 {
		return ""
	}

	ad := vast.Ads[0]

	// Try InLine first
	if ad.InLine != nil && ad.InLine.Creatives != nil {
		for _, creative := range ad.InLine.Creatives.Creative {
			if creative.Linear != nil && creative.Linear.Duration != "" {
				return creative.Linear.Duration
			}
		}
	}

	// Try Wrapper
	if ad.Wrapper != nil && ad.Wrapper.Creatives != nil {
		for _, creative := range ad.Wrapper.Creatives.Creative {
			if creative.Linear != nil && creative.Linear.Duration != "" {
				return creative.Linear.Duration
			}
		}
	}

	return ""
}

// ParseDurationToSeconds parses a VAST duration string (HH:MM:SS or HH:MM:SS.mmm) to seconds.
// Returns 0 if the duration cannot be parsed.
func ParseDurationToSeconds(duration string) int {
	if duration == "" {
		return 0
	}

	// Handle HH:MM:SS.mmm format (strip milliseconds)
	if idx := strings.Index(duration, "."); idx != -1 {
		duration = duration[:idx]
	}

	parts := strings.Split(duration, ":")
	if len(parts) != 3 {
		return 0
	}

	var hours, minutes, seconds int
	if _, err := parseIntFromString(parts[0], &hours); err != nil {
		return 0
	}
	if _, err := parseIntFromString(parts[1], &minutes); err != nil {
		return 0
	}
	if _, err := parseIntFromString(parts[2], &seconds); err != nil {
		return 0
	}

	return hours*3600 + minutes*60 + seconds
}

// parseIntFromString is a helper to parse an integer from a string.
func parseIntFromString(s string, result *int) (bool, error) {
	s = strings.TrimSpace(s)
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, errors.New("invalid character in number")
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return true, nil
}

// IsInLineAd returns true if the ad is an InLine ad (not a Wrapper).
func IsInLineAd(ad *Ad) bool {
	return ad != nil && ad.InLine != nil
}

// IsWrapperAd returns true if the ad is a Wrapper ad.
func IsWrapperAd(ad *Ad) bool {
	return ad != nil && ad.Wrapper != nil
}
