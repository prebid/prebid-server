package pubxai

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func createMockServer(statusCode int, responseBody string) *httptest.Server {
	// Create a new mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the status code
		w.WriteHeader(statusCode)
		// Write the response body
		w.Write([]byte(responseBody))
	}))
	return server
}

func TestFetchConfig_Success(t *testing.T) {
	// Mock HTTP server
	mockServer := createMockServer(http.StatusOK, `{"publisher_id": "test_publisher", "buffer_interval": "10s", "buffer_size": "10MB", "sampling_percentage": 50}`)
	defer mockServer.Close()

	// Create a new HTTP client with mock server URL
	client := mockServer.Client()
	endpointUrl, _ := url.Parse(mockServer.URL)

	// Call fetchConfig with the mock client and endpoint URL
	config, err := fetchConfig(client, endpointUrl)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Assert fetched configuration values
	if config.PublisherId != "test_publisher" {
		t.Errorf("Expected PublisherId to be 'test_publisher', got %s", config.PublisherId)
	}
	if config.BufferInterval != "10s" {
		t.Errorf("Expected BufferInterval to be '10s', got %s", config.BufferInterval)
	}
	// Add assertions for BufferSize and SamplingPercentage
}

func TestFetchConfig_HTTPError(t *testing.T) {

	mockServer := createMockServer(http.StatusNotFound, "")
	defer mockServer.Close()

	client := mockServer.Client()
	endpointUrl, _ := url.Parse(mockServer.URL)

	_, err := fetchConfig(client, endpointUrl)
	if err == nil {
		t.Error("Expected an error, got nil")
	}
}

func TestNewConfigUpdateHttpTask_Success(t *testing.T) {

	httpClient := &http.Client{}
	task, err := NewConfigUpdateHttpTask(httpClient, "test_pubxId", "http://example.com", "10s")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if task == nil {
		t.Error("Expected a non-nil task, got nil")
	}
}

func TestNewConfigUpdateHttpTask_InvalidRefreshInterval(t *testing.T) {

	httpClient := &http.Client{}
	_, err := NewConfigUpdateHttpTask(httpClient, "test_pubxId", "http://example.com", "invalid")
	if err == nil {
		t.Error("Expected an error, got nil")
	}

}

func TestIsSameAs_SameConfig(t *testing.T) {
	config1 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}
	config2 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}

	if !config1.isSameAs(config2) {
		t.Errorf("Expected configurations to be considered the same, but they are not")
	}
}

func TestIsSameAs_DifferentPublisherId(t *testing.T) {
	config1 := &Configuration{PublisherId: "test1", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}
	config2 := &Configuration{PublisherId: "test2", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}

	if config1.isSameAs(config2) {
		t.Errorf("Expected configurations to be considered different, but they are the same")
	}
}

func TestIsSameAs_DifferentBufferSize(t *testing.T) {
	config1 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}
	config2 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "20MB", SamplingPercentage: 50}

	if config1.isSameAs(config2) {
		t.Errorf("Expected configurations to be considered different, but they are the same")
	}
}

func TestIsSameAs_DifferentSamplingPercentage(t *testing.T) {
	config1 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 50}
	config2 := &Configuration{PublisherId: "test", BufferInterval: "10s", BufferSize: "10MB", SamplingPercentage: 75}

	if config1.isSameAs(config2) {
		t.Errorf("Expected configurations to be considered different, but they are the same")
	}
}
