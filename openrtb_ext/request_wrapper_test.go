package openrtb_ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Some minimal tests to get code coverage above 30%. The real tests are when other modules use these structures.

func TestUserExt(t *testing.T) {
	userExt := &UserExt{}

	userExt.unmarshal(nil)
	assert.Equal(t, false, userExt.Dirty(), "New UserExt should not be dirty.")
	assert.Nil(t, userExt.GetConsent(), "Empty UserExt should have nil consent")

	newConsent := "NewConsent"
	userExt.SetConsent(&newConsent)
	assert.Equal(t, "NewConsent", *userExt.GetConsent())

}
