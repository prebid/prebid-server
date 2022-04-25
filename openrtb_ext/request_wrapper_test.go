package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Some minimal tests to get code coverage above 30%. The real tests are when other modules use these structures.

func TestUserExt(t *testing.T) {
	userExt := &UserExt{}

	userExt.unmarshal(nil)
	assert.Equal(t, false, userExt.Dirty(), "New UserExt should not be dirty.")
	assert.Nil(t, userExt.GetConsent(), "Empty UserExt should have nil consent")
	assert.Nil(t, userExt.GetEid(), "Empty UserExt should have nil eid")
	assert.Nil(t, userExt.GetPrebid(), "Empty UserExt should have nil prebid")

	newConsent := "NewConsent"
	userExt.SetConsent(&newConsent)
	assert.Equal(t, "NewConsent", *userExt.GetConsent(), "UserExt consent is incorrect")

	newEid := []ExtUserEid{{}}
	userExt.SetEid(&newEid)
	assert.Equal(t, []ExtUserEid{{}}, *userExt.GetEid(), "UserExt eid is incorrect")

	buyerIDs := map[string]string{"buyer": "id"}
	newPrebid := ExtUserPrebid{BuyerUIDs: buyerIDs}
	userExt.SetPrebid(&newPrebid)
	assert.Equal(t, ExtUserPrebid{BuyerUIDs: buyerIDs}, *userExt.GetPrebid(), "UserExt prebid is icorrect")

	assert.Equal(t, true, userExt.Dirty(), "UserExt should be dirty after field updates")

	updatedUserExt, err := userExt.marshal()
	assert.Nil(t, err, "Marshalling UserExt after updating should not cause an error")

	expectedUserExt := json.RawMessage(`{"consent":"NewConsent","prebid":{"buyeruids":{"buyer":"id"}},"eids":[{"source":""}]}`)
	assert.JSONEq(t, string(updatedUserExt), string(expectedUserExt), "Marshalled UserExt is incorrect")

	assert.Equal(t, false, userExt.Dirty(), "UserExt should not be dirty after marshalling")
}
