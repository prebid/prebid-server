package prebid_cache_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/metrics"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"golang.org/x/net/context/ctxhttp"
)

// Client stores values in Prebid Cache. For more info, see https://github.com/prebid/prebid-cache
type Client interface {
	// PutJson stores JSON values for the given openrtb2.Bids in the cache. Null values will be
	//
	// The returned string slice will always have the same number of elements as the values argument. If a
	// value could not be saved, the element will be an empty string. Implementations are responsible for
	// logging any relevant errors to the app logs
	PutJson(ctx context.Context, values []Cacheable) ([]string, []error)

	// GetExtCacheData gets the scheme, host, and path of the externally accessible cache url.
	GetExtCacheData() (scheme string, host string, path string)
}

type PayloadType string

const (
	TypeJSON PayloadType = "json"
	TypeXML  PayloadType = "xml"
)

type Cacheable struct {
	Type       PayloadType     `json:"type,omitempty"`
	Data       json.RawMessage `json:"value,omitempty"`
	TTLSeconds int64           `json:"ttlseconds,omitempty"`
	Key        string          `json:"key,omitempty"`

	BidID     string `json:"bidid,omitempty"`     // this is "/vtrack" specific
	Bidder    string `json:"bidder,omitempty"`    // this is "/vtrack" specific
	Timestamp int64  `json:"timestamp,omitempty"` // this is "/vtrack" specific
}

func NewClient(httpClient *http.Client, conf *config.Cache, extCache *config.ExternalCache, metrics metrics.MetricsEngine) Client {
	return &clientImpl{
		httpClient:          httpClient,
		putUrl:              conf.GetBaseURL() + "/cache",
		externalCacheScheme: extCache.Scheme,
		externalCacheHost:   extCache.Host,
		externalCachePath:   extCache.Path,
		metrics:             metrics,
	}
}

type clientImpl struct {
	httpClient          *http.Client
	putUrl              string
	externalCacheScheme string
	externalCacheHost   string
	externalCachePath   string
	metrics             metrics.MetricsEngine
}

func (c *clientImpl) GetExtCacheData() (string, string, string) {
	path := c.externalCachePath
	if path == "/" {
		// Only the slash for the path, remove it to empty
		path = ""
	} else if len(path) > 0 && !strings.HasPrefix(path, "/") {
		// Path defined but does not start with "/", prepend it
		path = "/" + path
	}

	return c.externalCacheScheme, c.externalCacheHost, path
}

func (c *clientImpl) PutJson(ctx context.Context, values []Cacheable) (uuids []string, errs []error) {
	errs = make([]error, 0, 1)
	if len(values) < 1 {
		return nil, errs
	}

	uuidsToReturn := make([]string, len(values))

	postBody, err := encodeValues(values)
	if err != nil {
		logError(&errs, "Error creating JSON for prebid cache: %v", err)
		return uuidsToReturn, errs
	}

	httpReq, err := http.NewRequest("POST", c.putUrl, bytes.NewReader(postBody))
	if err != nil {
		logError(&errs, "Error creating POST request to prebid cache: %v", err)
		return uuidsToReturn, errs
	}

	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	startTime := time.Now()
	anResp, err := ctxhttp.Do(ctx, c.httpClient, httpReq)
	elapsedTime := time.Since(startTime)
	if err != nil {
		c.metrics.RecordPrebidCacheRequestTime(false, elapsedTime)
		logError(&errs, "Error sending the request to Prebid Cache: %v; Duration=%v, Items=%v, Payload Size=%v", err, elapsedTime, len(values), len(postBody))
		return uuidsToReturn, errs
	}
	defer anResp.Body.Close()
	c.metrics.RecordPrebidCacheRequestTime(true, elapsedTime)

	responseBody, err := ioutil.ReadAll(anResp.Body)
	if anResp.StatusCode != 200 {
		logError(&errs, "Prebid Cache call to %s returned %d: %s", putURL, anResp.StatusCode, responseBody)
		return uuidsToReturn, errs
	}

	currentIndex := 0
	processResponse := func(uuidObj []byte, _ jsonparser.ValueType, _ int, err error) {
		if uuid, valueType, _, err := jsonparser.Get(uuidObj, "uuid"); err != nil {
			logError(&errs, "Prebid Cache returned a bad value at index %d. Error was: %v. Response body was: %s", currentIndex, err, string(responseBody))
		} else if valueType != jsonparser.String {
			logError(&errs, "Prebid Cache returned a %v at index %d in: %v", valueType, currentIndex, string(responseBody))
		} else {
			if uuidsToReturn[currentIndex], err = jsonparser.ParseString(uuid); err != nil {
				logError(&errs, "Prebid Cache response index %d could not be parsed as string: %v", currentIndex, err)
				uuidsToReturn[currentIndex] = ""
			}
		}
		currentIndex++
	}

	if _, err := jsonparser.ArrayEach(responseBody, processResponse, "responses"); err != nil {
		logError(&errs, "Error interpreting Prebid Cache response: %v\nResponse was: %s", err, string(responseBody))
		return uuidsToReturn, errs
	}

	return uuidsToReturn, errs
}

func logError(errs *[]error, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	glog.Error(msg)
	*errs = append(*errs, errors.New(msg))
}

func encodeValues(values []Cacheable) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"puts":[`)
	for i := 0; i < len(values); i++ {
		if err := encodeValueToBuffer(values[i], i != 0, &buf); err != nil {
			return nil, err
		}
	}
	buf.WriteString("]}")
	return buf.Bytes(), nil
}

func encodeValueToBuffer(value Cacheable, leadingComma bool, buffer *bytes.Buffer) error {
	if leadingComma {
		buffer.WriteByte(',')
	}

	buffer.WriteString(`{"type":"`)
	buffer.WriteString(string(value.Type))
	if value.TTLSeconds > 0 {
		buffer.WriteString(`","ttlseconds":`)
		buffer.WriteString(strconv.FormatInt(value.TTLSeconds, 10))
		buffer.WriteString(`,"value":`)
	} else {
		buffer.WriteString(`","value":`)
	}
	buffer.Write(value.Data)
	if len(value.Key) > 0 {
		buffer.WriteString(`,"key":"`)
		buffer.WriteString(string(value.Key))
		buffer.WriteString(`"`)
	}

	//vtrack specific
	if len(value.BidID) > 0 {
		buffer.WriteString(`,"bidid":"`)
		buffer.WriteString(string(value.BidID))
		buffer.WriteString(`"`)
	}

	if len(value.Bidder) > 0 {
		buffer.WriteString(`,"bidder":"`)
		buffer.WriteString(string(value.Bidder))
		buffer.WriteString(`"`)
	}

	if value.Timestamp > 0 {
		buffer.WriteString(`,"timestamp":`)
		buffer.WriteString(strconv.FormatInt(value.Timestamp, 10))
	}

	buffer.WriteByte('}')
	return nil
}
