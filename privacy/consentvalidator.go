package privacy

type ConsentValidator interface {
	ValidateConsent(consent string) bool
}

// MockConsentValidator implements ConsentValidator and is useful for testing purposes
type MockConsentValidator struct {
	ReturnValue bool
}

func (cv MockConsentValidator) ValidateConsent(consent string) bool {
	return cv.ReturnValue
}
