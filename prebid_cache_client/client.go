package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
)

// Client stores values in Prebid Cache. For more info, see https://github.com/prebid/prebid-cache
type Client interface {
	// PutBids stores JSON values for the given openrtb.Bids in the cache. values can be nil, but
	// must not contain nil elements.
	//
	// The returned string slice will always have the same number of elements as the values argument. If a
	// value could not be saved, the element will be an empty string. Implementations are responsible for
	// logging any relevant errors to the app logs
	PutBids(ctx context.Context, values []*openrtb.Bid) []string
}

func NewClient(conf *config.Cache) Client {
	return &clientImpl{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 65,
			},
		},
		putUrl: baseURL + "/cache",
	}
}

type clientImpl struct {
	httpClient *http.Client
	putUrl     string
}

func (c *clientImpl) PutBids(ctx context.Context, values []*openrtb.Bid) (uuids []string) {
	if values == nil || len(values) < 1 {
		return nil
	}

	uuidsToReturn := make([]string, len(values))
	postBody, err := marshalBidList(values)
	if err != nil {
		glog.Errorf("Error marshalling bids for prebid cache request: %v", err)
		return uuidsToReturn
	}

	httpReq, err := http.NewRequest("POST", c.putUrl, bytes.NewReader(postBody))
	if err != nil {
		glog.Errorf("Error creating POST request to prebid cache: %v", err)
		return uuidsToReturn
	}
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, c.httpClient, httpReq)
	if err != nil {
		glog.Errorf("Error sending the request to Prebid Cache: %v", err)
		return uuidsToReturn
	}
	defer anResp.Body.Close()
	responseBody, err := ioutil.ReadAll(anResp.Body)

	if anResp.StatusCode != 200 {
		glog.Errorf("Prebid Cache returned %d: %v", anResp.StatusCode, err)
		return uuidsToReturn
	}

	currentIndex := 0
	processResponse := func(uuidObj []byte, dataType jsonparser.ValueType, offset int, err error) {
		if uuid, valueType, _, err := jsonparser.Get(uuidObj, "uuid"); err != nil {
			glog.Errorf("Prebid Cache returned a bad value at index %d. Error was: %v. Response body was: %s", currentIndex, err, string(responseBody))
		} else if valueType != jsonparser.String {
			glog.Errorf("Prebid Cache returned a %v at index %d in: %v", valueType, currentIndex, string(responseBody))
		} else {
			if uuidsToReturn[currentIndex], err = jsonparser.ParseString(uuid); err != nil {
				glog.Errorf("Prebid Cache response index %d could not be parsed as string: %v", currentIndex, err)
				uuidsToReturn[currentIndex] = ""
			}
		}
		currentIndex++
	}

	if _, err := jsonparser.ArrayEach(responseBody, processResponse, "responses"); err != nil {
		glog.Errorf("Error interpreting Prebid Cache response: %v\nResponse was: %s", err, string(responseBody))
		return uuidsToReturn
	}

	return uuidsToReturn
}

// marshalBidList encodes an []*openrtb.Bid into JSON for the Prebid Cache API.
func marshalBidList(bids []*openrtb.Bid) ([]byte, error) {
	// This function assumes that m is non-nil and has at least one element.
	// clientImp.PutBids should respect this.
	var buf bytes.Buffer
	buf.WriteString(`{"puts":[`)
	for i := 0; i < len(bids); i++ {
		if err := marshalBidToBuffer(bids[i], i != 0, &buf); err != nil {
			return nil, err
		}
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

// marshalBidToBuffer writes JSON for bid into the buffer, with a leading comma if necessary.
func marshalBidToBuffer(bid *openrtb.Bid, leadingComma bool, buffer *bytes.Buffer) error {
	if leadingComma {
		buffer.WriteByte(',')
	}
	if bid == nil {
		buffer.WriteString("null")
		return nil
	}

	bidJson, err := json.Marshal(bid)
	if err != nil {
		return err
	}
	buffer.WriteString(`{"type":"json","value":`)
	buffer.Write(bidJson)
	buffer.WriteByte('}')
	return nil
}
