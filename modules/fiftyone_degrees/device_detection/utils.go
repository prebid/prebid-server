package device_detection

// Contains checks if a string is in a slice of strings
func Contains(source []string, item string) bool {
	for _, element := range source {
		if item == element {
			return true
		}
	}
	return false
}
