package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalPublisher(t *testing.T) {
	test1 := `{
       "id": "publisherID",
		"name": "publisherName",
		"domain": "www.mydomain.org",
		"ext": {
			"prebid": {
				"parentAccount": "anAccountId"
			}
		}
	}
`
	var err error

	var publisherObj openrtb.Publisher
	err = json.Unmarshal([]byte(test1), &publisherObj)
	assert.NoError(t, err, "[extPublisherPrebidObj] Error Unmarshaling Publisher\n")
	assert.NotNil(t, publisherObj, "[extPublisherPrebidObj] Error unmarshaled publisherObj should not evaluate to NULL \n")

	var extPublisherObj ExtPublisher
	err = json.Unmarshal(publisherObj.Ext, &extPublisherObj)

	assert.NoError(t, err, "[extPublisherPrebidObj] Error Unmarshaling Publisher.Ext. \n")
	assert.NotNil(t, extPublisherObj.Prebid, "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount. extPublisherObj.Prebid should not be nil\n")
	assert.NotNil(t, extPublisherObj.Prebid.ParentAccount, "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount. extPublisherObj.Prebid.ParentAccount should not be nil\n")
	assert.Equal(t, *extPublisherObj.Prebid.ParentAccount, "anAccountId", "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount.\n")
}

func TestUnmarshalExtPublisher(t *testing.T) {
	test2 := &openrtb.Publisher{
		ID:     "publisherID",
		Name:   "publisherName",
		Domain: "www.mydomain.org",
		Ext: json.RawMessage(`{
	"prebid": {
		"parentAccount": "anAccountId"
	}
}
`),
	}
	var err error

	var extPublisherObj ExtPublisher
	err = json.Unmarshal([]byte(test2.Ext), &extPublisherObj)

	assert.NoError(t, err, "[extPublisherPrebidObj] Error Unmarshaling Publisher.Ext. \n")
	assert.NotNil(t, extPublisherObj, "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount. extPublisherObj.Prebid should not be nil\n")
	assert.NotNil(t, extPublisherObj.Prebid.ParentAccount, "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount. extPublisherObj.Prebid.ParentAccount should not be nil\n")
	assert.Equal(t, *extPublisherObj.Prebid.ParentAccount, "anAccountId", "[extPublisherPrebidObj] Error reading publisher.ext.prebid.parentAccount.\n")
}
