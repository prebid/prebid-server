package iosutil

type IOSVersion struct {
	Major int
	Minor int
}

func ParseIOSVersion(v string) (IOSVersion, error) {
	return nil, nil
}

func (v IOSVersion) EqualOrGreater(major, minor int) bool {
	if v.Major == major {
		return v.Minor >= minor
	}

	return v.Major < major
}
