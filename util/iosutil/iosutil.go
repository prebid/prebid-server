package iosutil

import (
	"errors"
	"strconv"
	"strings"
)

// IOSVersion specifies the version of an iOS device.
type IOSVersion struct {
	Major int
	Minor int
}

// ParseIOSVersion parses the major.minor version for an iOS device.
func ParseIOSVersion(v string) (IOSVersion, error) {
	parts := strings.Split(v, ".")

	if len(parts) != 2 {
		return IOSVersion{}, errors.New("expected major.minor format")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return IOSVersion{}, errors.New("major version is not an integer")
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return IOSVersion{}, errors.New("minor version is not an integer")
	}

	version := IOSVersion{
		Major: major,
		Minor: minor,
	}
	return version, nil
}

// EqualOrGreater returns true if iOS device version is equal or greater to the desired version, using semantic versioning.
func (v IOSVersion) EqualOrGreater(major, minor int) bool {
	if major == v.Major {
		return minor >= v.Minor
	}

	return v.Major < major
}
