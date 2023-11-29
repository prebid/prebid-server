package currency

import (
	"io"
	"net/http"
	"strings"
)

// MockCurrencyRatesHttpClient is a simple http client mock returning a constant response body
type MockCurrencyRatesHttpClient struct {
	ResponseBody string
}

func (m *MockCurrencyRatesHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(m.ResponseBody)),
	}, nil
}
