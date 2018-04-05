package http_fetcher

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Tests for buildRequest
func TestSingleReq(t *testing.T) {
	doBuildURLTest(t, "http://prebid.com/stored_requests", []string{"req-1"}, nil, "http://prebid.com/stored_requests?req-ids=req-1")
}

func TestSeveralReqs(t *testing.T) {
	doBuildURLTest(t, "http://prebid.com/stored_requests", []string{"req-1", "req-2"}, nil, "http://prebid.com/stored_requests?req-ids=req-1,req-2")
}

func TestSingleImp(t *testing.T) {
	doBuildURLTest(t, "http://prebid.com/stored_requests", nil, []string{"imp-1"}, "http://prebid.com/stored_requests?imp-ids=imp-1")
}

func TestSeveralImps(t *testing.T) {
	doBuildURLTest(t, "http://prebid.com/stored_requests", nil, []string{"imp-1", "imp-2"}, "http://prebid.com/stored_requests?imp-ids=imp-1,imp-2")
}

func TestReqsAndImps(t *testing.T) {
	doBuildURLTest(t, "http://prebid.com/stored_requests", []string{"req-1"}, []string{"imp-1"}, "http://prebid.com/stored_requests?req-ids=req-1&imp-ids=imp-1")
}

// Tests for unpackResponse
func TestReqResp(t *testing.T) {
	doResponseUnpackTest(t, `{"requests":{"req-1":{"isRequest":true}}}`, map[string]json.RawMessage{"req-1": json.RawMessage(`{"isRequest":true}`)}, nil, nil)
}

func TestImpResp(t *testing.T) {
	doResponseUnpackTest(t, `{"imps":{"imp-1":{"isRequest":false}}}`, nil, map[string]json.RawMessage{"imp-1": json.RawMessage(`{"isRequest":false}`)}, nil)
}

func TestImpReqResp(t *testing.T) {
	mockResponse := `{"requests":{"req-1":{"isRequest":true}},"imps":{"imp-1":{"isRequest":false}}}`
	expectedRequestData := map[string]json.RawMessage{"req-1": json.RawMessage(`{"isRequest":true}`)}
	expectedImpData := map[string]json.RawMessage{"imp-1": json.RawMessage(`{"isRequest":false}`)}
	doResponseUnpackTest(t, mockResponse, expectedRequestData, expectedImpData, nil)
}

func TestMalformedResponse(t *testing.T) {
	doResponseUnpackTest(t, `{`, nil, nil, []string{"unexpected end of JSON input"})
}

func TestErrorResponse(t *testing.T) {
	mockResponse := &http.Response{
		StatusCode: 502,
		Body:       closeWrapper{strings.NewReader("Bad response")},
	}
	requestData, impData, errs := unpackResponse(mockResponse)
	if len(requestData) > 0 {
		t.Errorf("Bad requestData length: %d", len(requestData))
	}
	if len(impData) > 0 {
		t.Errorf("Bad impData length: %d", len(impData))
	}
	if len(errs) != 1 {
		t.Fatalf("Bad err length: %d", len(errs))
	}
}

func doBuildURLTest(t *testing.T, endpoint string, requests []string, imps []string, expected string) {
	httpFetcher := NewFetcher(nil, endpoint)
	req, err := buildRequest(httpFetcher.endpoint, requests, imps)
	if err != nil {
		t.Fatalf("Unexpected error building URL: %v", err)
	}

	if req.URL.String() != expected {
		t.Errorf("Bad URL. Expected %s, got %s", expected, req.URL.String())
	}
}

func doResponseUnpackTest(t *testing.T, resp string, expectedReqs map[string]json.RawMessage, expectedImps map[string]json.RawMessage, expectedErrs []string) {
	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       closeWrapper{strings.NewReader(resp)},
	}

	requestData, impData, errs := unpackResponse(mockResponse)
	assertSameContents(t, requestData, expectedReqs)
	assertSameContents(t, impData, expectedImps)
	assertSameErrMsgs(t, expectedErrs, errs)
}

func assertSameContents(t *testing.T, expected map[string]json.RawMessage, actual map[string]json.RawMessage) {
	if len(expected) != len(actual) {
		t.Errorf("Wrong counts. Expected %d, actual %d", len(expected), len(actual))
		return
	}
	for expectedKey, expectedVal := range expected {
		if actualVal, ok := actual[expectedKey]; ok {
			if !bytes.Equal(expectedVal, actualVal) {
				t.Errorf("actual[%s] value %s does not match expected: %s", expectedKey, string(actualVal), string(actualVal))
			}
		} else {
			t.Errorf("actual map missing expected key %s", expectedKey)
		}
	}
}

func assertSameErrMsgs(t *testing.T, expected []string, actual []error) {
	if len(expected) != len(actual) {
		t.Errorf("Wrong error counts. Expected %d, actual %d", len(expected), len(actual))
		return
	}
	for i, expectedErr := range expected {
		if actual[i].Error() != expectedErr {
			t.Errorf("Wrong error[%d]. Expected %s, got %s", i, expectedErr, actual[i].Error())
		}
	}
}

type closeWrapper struct {
	io.Reader
}

func (w closeWrapper) Close() error {
	return nil
}
