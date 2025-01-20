package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/util/task"
)

type Configuration struct {
	PublisherId        string `json:"publisher_id"`
	BufferInterval     string `json:"buffer_interval"`
	BufferSize         string `json:"buffer_size"`
	SamplingPercentage int    `json:"sampling_percentage"`
}

type ConfigService interface {
	Start(stop <-chan struct{}) <-chan *Configuration
	IsSameAs(a *Configuration, b *Configuration) bool
}

type ConfigServiceImpl struct {
	task       *task.TickerTask
	configChan chan *Configuration
}

func NewConfigService(httpClient *http.Client, pubxId, endpoint, refreshInterval string) (ConfigService, error) {
	refreshDuration, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubxai.configuration_refresh_delay: %v", err)
	}
	endpointUrl, err := url.Parse(endpoint + "/config")
	if err != nil {
		return nil, err
	}

	query := endpointUrl.Query()
	query.Set("pubxId", pubxId)
	endpointUrl.RawQuery = query.Encode()

	configChan := make(chan *Configuration)

	tr := task.NewTickerTaskFromFunc(refreshDuration, func() error {
		config, err := fetchConfig(httpClient, endpointUrl)
		if err != nil {
			return fmt.Errorf("[pubxai] Fail to fetch remote configuration: %v", err)
		}
		configChan <- config
		return nil
	})

	return &ConfigServiceImpl{
		task:       tr,
		configChan: configChan,
	}, nil
}

func fetchConfig(client *http.Client, endpoint *url.URL) (*Configuration, error) {
	res, err := client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(&c)
	glog.Info("[pubxai] fetchConfig: %v at time %v", c, time.Now())
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (t *ConfigServiceImpl) Start(stop <-chan struct{}) <-chan *Configuration {
	go t.task.Start()

	go func() {
		<-stop
		t.task.Stop()
	}()

	return t.configChan
}

func (t *ConfigServiceImpl) IsSameAs(a *Configuration, b *Configuration) bool {

	samePublisherId := a.PublisherId == b.PublisherId
	sameBufferInterval := a.BufferInterval == b.BufferInterval
	sameBufferSize := a.BufferSize == b.BufferSize
	sameSamplingPercentage := a.SamplingPercentage == b.SamplingPercentage
	return samePublisherId && sameBufferInterval && sameBufferSize && sameSamplingPercentage
}
