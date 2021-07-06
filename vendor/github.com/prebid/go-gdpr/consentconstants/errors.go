package consentconstants

import "errors"

var (
	// ErrEmptyDecodedConsent error raised when the consent string is empty
	ErrEmptyDecodedConsent = errors.New("decoded consent cannot be empty")
)
