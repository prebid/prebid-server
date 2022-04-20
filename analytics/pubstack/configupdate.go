package pubstack

import (
	"fmt"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/util/task"
)

// ConfigUpdateTask defines a process which publishes Pubstack configurations until the stop channel is signaled.
type ConfigUpdateTask interface {
	Start(stop <-chan struct{}) <-chan *Configuration
}

// ConfigUpdateHttpTask polls an HTTP endpoint on a specified refresh interval and publishes configurations until
// the stop channel is signaled.
type ConfigUpdateHttpTask struct {
	task task.TickerTask
}

func (t *ConfigUpdateHttpTask) Start(stop <-chan struct{}) <-chan *Configuration {
	t.task.Start()

	go func() {
		<-stop
		t.task.Stop()
	}()
}

func NewConfigUpdateHttpTask(endpoint, scope, refreshInterval string) (*ConfigUpdateTask, error) {
	interval, err := time.ParseDuration(refreshInterval)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubstack.configuration_refresh_delay, :%v", err)
	}

	endpointUrl, err := url.Parse(endpoint + "/bootstrap?scopeId=" + scope)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	tr := task.NewTickerTaskFromMethod(interval, taskA.reloadConfig)

	return ConfigUpdateHttpTask{
		httpClient: nil,
		configCh:   make(chan *Configuration),
		endpoint:   endpointUrl,
		sigTermCh:  nil,
	}, nil
}

func (t *ConfigUpdateHttpTask) reloadConfig() {
	config, err := fetchConfig(t.httpClient, t.endpoint)

	if err != nil {
		glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
		return
	}

	t.configCh <- config

	if !chanOpen {
		t.task.Stop()
	}
}
