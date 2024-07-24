package device_detection

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/stretchr/testify/assert"
)

func TestAccountValidatorAllowed(t *testing.T) {
	validator := NewAccountValidator()
	cfg := Config{
		AccountFilter: AccountFilter{AllowList: []string{"1001"}},
	}

	res := validator.IsAllowed(
		cfg, toBytes(
			&openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "1001",
					},
				},
			},
		),
	)
	assert.True(t, res)
}

func TestAllowedAccountsIsEmpty(t *testing.T) {
	validator := NewAccountValidator()
	cfg := Config{
		AccountFilter: AccountFilter{AllowList: []string{}},
	}

	res := validator.IsAllowed(
		cfg, toBytes(
			&openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "1001",
					},
				},
			},
		),
	)
	assert.True(t, res)
}

func TestAccountValidatorNotAllowed(t *testing.T) {
	validator := NewAccountValidator()
	cfg := Config{
		AccountFilter: AccountFilter{AllowList: []string{"1002"}},
	}

	res := validator.IsAllowed(
		cfg, toBytes(
			&openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "1001",
					},
				},
			},
		),
	)
	assert.False(t, res)
}

func toBytes(v interface{}) []byte {
	res, _ := json.Marshal(v)
	return res
}
