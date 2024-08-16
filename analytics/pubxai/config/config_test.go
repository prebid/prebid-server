package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigService(t *testing.T) {
	httpClient := &http.Client{}
	pubxId := "testPublisher"
	endpoint := "http://example.com"
	refreshInterval := "1m"

	configService, err := NewConfigService(httpClient, pubxId, endpoint, refreshInterval)
	assert.NoError(t, err)
	assert.NotNil(t, configService)
}

func TestNewConfigService_InvalidDuration(t *testing.T) {
	httpClient := &http.Client{}
	pubxId := "testPublisher"
	endpoint := "http://example.com"
	refreshInterval := "invalid"

	configService, err := NewConfigService(httpClient, pubxId, endpoint, refreshInterval)
	assert.Error(t, err)
	assert.Nil(t, configService)
}

func TestFetchConfig(t *testing.T) {
	expectedConfig := &Configuration{
		PublisherId:        "testPublisher",
		BufferInterval:     "30s",
		BufferSize:         "100",
		SamplingPercentage: 50,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedConfig)
	}))
	defer server.Close()

	endpointUrl, _ := url.Parse(server.URL)

	config, err := fetchConfig(server.Client(), endpointUrl)
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}

func TestFetchConfig_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	endpointUrl, _ := url.Parse(server.URL)

	config, err := fetchConfig(server.Client(), endpointUrl)
	assert.Error(t, err)
	assert.Nil(t, config)
}

func TestConfigServiceImpl_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	httpClient := &http.Client{}
	pubxId := "testPublisher"
	endpoint := "http://example.com"
	refreshInterval := "1m"

	configService, _ := NewConfigService(httpClient, pubxId, endpoint, refreshInterval)
	configImpl := configService.(*ConfigServiceImpl)

	stop := make(chan struct{})
	configChan := configImpl.Start(stop)

	// Ensure task starts correctly
	time.Sleep(2 * time.Second)
	close(stop)
	// Ensure task stops correctly
	time.Sleep(2 * time.Second)
	assert.NotNil(t, configChan)
}

func TestConfigServiceImpl_IsSameAs(t *testing.T) {
	config1 := &Configuration{
		PublisherId:        "testPublisher",
		BufferInterval:     "30s",
		BufferSize:         "100",
		SamplingPercentage: 50,
	}

	config2 := &Configuration{
		PublisherId:        "testPublisher",
		BufferInterval:     "30s",
		BufferSize:         "100",
		SamplingPercentage: 50,
	}

	config3 := &Configuration{
		PublisherId:        "differentPublisher",
		BufferInterval:     "30s",
		BufferSize:         "100",
		SamplingPercentage: 50,
	}

	configService := &ConfigServiceImpl{}

	assert.True(t, configService.IsSameAs(config1, config2))
	assert.False(t, configService.IsSameAs(config1, config3))
}
