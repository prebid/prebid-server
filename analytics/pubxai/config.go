package pubxai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/util/task"
)

type ConfigUpdateTask interface {
	Start(stop <-chan struct{}) <-chan *Configuration
}
type ConfigUpdateHttpTask struct {
	task       *task.TickerTask
	configChan chan *Configuration
}

func fetchConfig(client *http.Client, endpoint *url.URL) (*Configuration, error) {
	res, err := client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(&c)
	glog.Info("fetchConfig: %v at time %v", c, time.Now())
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func NewConfigUpdateHttpTask(httpClient *http.Client, pubxId, endpoint, refreshInterval string) (*ConfigUpdateHttpTask, error) {
	refreshDuration, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubxai.configuration_refresh_delay: %v", err)
	}

	endpointUrl, err := url.Parse(endpoint + "/config?pubxId=" + pubxId)
	if err != nil {
		return nil, err
	}

	configChan := make(chan *Configuration)

	tr := task.NewTickerTaskFromFunc(refreshDuration, func() error {
		config, err := fetchConfig(httpClient, endpointUrl)
		if err != nil {
			return fmt.Errorf("[pubxai] Fail to fetch remote configuration: %v", err)
		}
		configChan <- config
		return nil
	})

	return &ConfigUpdateHttpTask{
		task:       tr,
		configChan: configChan,
	}, nil
}

func (t *ConfigUpdateHttpTask) Start(stop <-chan struct{}) <-chan *Configuration {
	go t.task.Start()

	go func() {
		<-stop
		t.task.Stop()
	}()

	return t.configChan
}

func (a *Configuration) isSameAs(b *Configuration) bool {

	samePublisherId := a.PublisherId == b.PublisherId
	sameBufferInterval := a.BufferInterval == b.BufferInterval
	sameBufferSize := a.BufferSize == b.BufferSize
	sameSamplingPercentage := a.SamplingPercentage == b.SamplingPercentage
	return samePublisherId && sameBufferInterval && sameBufferSize && sameSamplingPercentage
}
