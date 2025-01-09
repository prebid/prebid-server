package pubstack

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/prebid/prebid-server/v3/util/task"
)

// ConfigUpdateTask publishes configurations until the stop channel is signaled.
type ConfigUpdateTask interface {
	Start(stop <-chan struct{}) <-chan *Configuration
}

// ConfigUpdateHttpTask polls an HTTP endpoint on a specified interval and publishes configurations until
// the stop channel is signaled.
type ConfigUpdateHttpTask struct {
	task       *task.TickerTask
	configChan chan *Configuration
}

func NewConfigUpdateHttpTask(httpClient *http.Client, scope, endpoint, refreshInterval string) (*ConfigUpdateHttpTask, error) {
	refreshDuration, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubstack.configuration_refresh_delay: %v", err)
	}

	endpointUrl, err := url.Parse(endpoint + "/bootstrap?scopeId=" + scope)
	if err != nil {
		return nil, err
	}

	configChan := make(chan *Configuration)

	tr := task.NewTickerTaskFromFunc(refreshDuration, func() error {
		config, err := fetchConfig(httpClient, endpointUrl)
		if err != nil {
			return fmt.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
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
