package vendorconsent

import (
	"encoding/base64"
	"testing"
)

func assertInvalid(t *testing.T, urlEncodedString string, expectError string) {
	t.Helper()
	data, err := base64.RawURLEncoding.DecodeString(urlEncodedString)
	assertNilError(t, err)
	assertInvalidBytes(t, data, expectError)
}

func assertInvalidBytes(t *testing.T, data []byte, expectError string) {
	t.Helper()
	if consent, err := Parse(data); err == nil {
		t.Errorf("base64 URL-encoded string %s was considered valid, but shouldn't be. MaxVendorID: %d. len(data): %d", base64.RawURLEncoding.EncodeToString(data), consent.MaxVendorID(), len(data))
	} else if err.Error() != expectError {
		t.Errorf(`error messages did not match. Expected "%s", got "%s": %v`, expectError, err.Error(), err)
	}
}

func decode(t *testing.T, encodedString string) []byte {
	data, err := base64.RawURLEncoding.DecodeString(encodedString)
	assertNilError(t, err)
	return data
}

func assertNilError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func assertStringsEqual(t *testing.T, expected string, actual string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Strings were not equal. Expected %s, actual %s", expected, actual)
	}
}

func assertUInt8sEqual(t *testing.T, expected uint8, actual uint8) {
	t.Helper()
	if actual != expected {
		t.Errorf("Ints were not equal. Expected %d, actual %d", expected, actual)
	}
}

func assertUInt16sEqual(t *testing.T, expected uint16, actual uint16) {
	t.Helper()
	if actual != expected {
		t.Errorf("Ints were not equal. Expected %d, actual %d", expected, actual)
	}
}

func assertIntsEqual(t *testing.T, expected int, actual int) {
	t.Helper()
	if actual != expected {
		t.Errorf("Ints were not equal. Expected %d, actual %d", expected, actual)
	}
}

func assertBoolsEqual(t *testing.T, expected bool, actual bool) {
	t.Helper()
	if actual != expected {
		t.Errorf("Bools were not equal. Expected %t, actual %t", expected, actual)
	}
}

func buildMap(keys ...uint) map[uint]struct{} {
	var s struct{}
	m := make(map[uint]struct{}, len(keys))
	for _, key := range keys {
		m[key] = s
	}
	return m
}
