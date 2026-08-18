package adocean

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schemas: %v", err)
	}

	for _, params := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdOcean, json.RawMessage(params)); err != nil {
			t.Errorf("Schema rejected valid params: %s", params)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schemas: %v", err)
	}

	for _, params := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdOcean, json.RawMessage(params)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", params)
		}
	}
}

var validParams = []string{
	`{"emitterPrefix":"myao","masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7","slaveId":"adoceanmyaozpniqismex"}`,
	`{"emitterPrefix":"myao-test","masterId":"master_id.1","slaveId":"adoceanmyaozpniqismex","emitterRequestParams":{"test_parameter":"1"}}`,
}

var invalidParams = []string{
	`{}`,
	`{"masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7","slaveId":"adoceanmyaozpniqismex"}`,
	`{"emitterPrefix":"myao.adocean.pl","masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7","slaveId":"adoceanmyaozpniqismex"}`,
	`{"emitterPrefix":"myao","masterId":"master/id","slaveId":"adoceanmyaozpniqismex"}`,
	`{"emitterPrefix":"myao","masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7","slaveId":"myaozpniqismex"}`,
	`{"emitterPrefix":"myao","masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7"}`,
	`{"emitterPrefix":"myao","masterId":"tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7","slaveId":"adoceanmyaozpniqismex","emitterRequestParams":["invalid"]}`,
}
