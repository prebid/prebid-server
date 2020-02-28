package forwarder

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics/pubstack/parser"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func mapFileToObject(path string, tg interface{}) error {
	fl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fl.Close()

	data, err := ioutil.ReadAll(fl)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), tg)
	if err == nil {
		return err
	}

	return nil
}

func TestForwarder(t *testing.T) {
	testRq := openrtb.BidRequest{}
	testRp := openrtb.BidResponse{}
	p := parser.NewParser("test-scope")

	err := mapFileToObject("mocks/mock_openrtb_request.json", &testRq)
	assert.Equal(t, err, nil)
	err = mapFileToObject("mocks/mock_openrtb_response.json", &testRp)
	assert.Equal(t, err, nil)

	ret := p.Feed(&testRq, &testRp)
	fw := NewForwarder("https://intake.dev.pubstack.io/v1/intake/auction")
	res := fw.Feed(ret)
	assert.Equal(t, nil, res)
}
