package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
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

	eid := openrtb2.EID{Source: "source", UIDs: []openrtb2.UID{{ID: "id"}}}
	newEid := []openrtb2.EID{eid}
	userExt.SetEid(&newEid)
	assert.Equal(t, []openrtb2.EID{eid}, *userExt.GetEid(), "UserExt eid is incorrect")

	buyerIDs := map[string]string{"buyer": "id"}
	newPrebid := ExtUserPrebid{BuyerUIDs: buyerIDs}
	userExt.SetPrebid(&newPrebid)
	assert.Equal(t, ExtUserPrebid{BuyerUIDs: buyerIDs}, *userExt.GetPrebid(), "UserExt prebid is incorrect")

	assert.Equal(t, true, userExt.Dirty(), "UserExt should be dirty after field updates")

	updatedUserExt, err := userExt.marshal()
	assert.Nil(t, err, "Marshalling UserExt after updating should not cause an error")

	expectedUserExt := json.RawMessage(`{"consent":"NewConsent","prebid":{"buyeruids":{"buyer":"id"}},"eids":[{"source":"source","uids":[{"id":"id"}]}]}`)
	assert.JSONEq(t, string(expectedUserExt), string(updatedUserExt), "Marshalled UserExt is incorrect")

	assert.Equal(t, false, userExt.Dirty(), "UserExt should not be dirty after marshalling")
}
