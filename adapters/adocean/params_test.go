package adocean

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdOcean, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adocean params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdOcean, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"emitterPrefix": "myao", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emiter": "myao.adocean.pl", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
}

var invalidParams = []string{
	`{}`,
	`{"emitterPrefix": "myao", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emitterPrefix": "myao", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7"}`,
	`{"emitterPrefix": "", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emitterPrefix": "myao", "", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emitterPrefix": "myao", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": ""}`,
	`{"emitterPrefix": "myao", "masterId": "tmYF.DMl7Z utQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emitterPrefix": "myao", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmy iqismex"}`,
	`{"emitterPrefix": "myao.adocean.pl", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emiter": "myao.adocean.pl", "slaveId": "adoceanmyaozpniqismex"}`,
	`{"emiter": "", "masterId": "tmYF.DMl7ZBq.Nqt2Bq4FutQTJfTpxCOmtNPZoQUDcL.G7", "slaveId": "adoceanmyaozpniqismex"}`,
}
