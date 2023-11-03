package iosutil

import (
	"errors"
	"strconv"
	"strings"
)

// Version specifies the version of an iOS device.
type Version struct {
	Major int
	Minor int
}

// ParseVersion parses the major.minor version for an iOS device.
func ParseVersion(v string) (Version, error) {
	parts := strings.Split(v, ".")

	if len(parts) < 2 || len(parts) > 3 {
		return Version{}, errors.New("expected either major.minor or major.minor.patch format")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, errors.New("major version is not an integer")
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, errors.New("minor version is not an integer")
	}

	version := Version{
		Major: major,
		Minor: minor,
	}
	return version, nil
}

// Equal returns true if the iOS device version is equal to the desired major and minor version, using semantic versioning.
func (v Version) Equal(major, minor int) bool {
	if v.Major == major {
		return v.Minor == minor
	}

	return false
}

// EqualOrGreater returns true if the iOS device version is equal or greater to the desired version, using semantic versioning.
func (v Version) EqualOrGreater(major, minor int) bool {
	if v.Major == major {
		return v.Minor >= minor
	}

	return v.Major > major
}

// VersionClassification describes iOS version classifications which are important to Prebid Server.
type VersionClassification int

// Values of VersionClassification.
const (
	VersionUnknown VersionClassification = iota
	Version140
	Version141
	Version142OrGreater
)

// DetectVersionClassification detects the iOS version classification.
func DetectVersionClassification(v string) VersionClassification {
	// semantic versioning comparison second. parsing required.
	if iosVersion, err := ParseVersion(v); err == nil {
		if iosVersion.Equal(14, 0) {
			return Version140
		}
		if iosVersion.Equal(14, 1) {
			return Version141
		}
		if iosVersion.EqualOrGreater(14, 2) {
			return Version142OrGreater
		}
	}
	return VersionUnknown
}
