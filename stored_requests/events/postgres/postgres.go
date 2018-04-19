package postgres

import (
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
)

func NewPostgresEvents(cfg *config.PostgresEventsConfig) (e events.EventProducer, shutdown func() error) {
	listener := pq.NewListener(cfg.ConnectionInfo.ConnString(), time.Duration(cfg.MinReconnectInterval)*time.Millisecond, time.Duration(cfg.MaxReconnectInterval)*time.Millisecond, nil)
	if err := listener.Listen(cfg.Channel); err != nil {
		glog.Fatalf("Postgres notifier falied to listen on channel %s: %v", cfg.Channel, err)
	}

	saves := make(chan events.Save, 10)
	invalidations := make(chan events.Invalidation, 10)

	go forwardNotifications(listener.NotificationChannel(), saves, invalidations)

	return &postgresEvents{
		saves:         saves,
		invalidations: invalidations,
	}, listener.Close
}

type postgresEvents struct {
	saves         <-chan events.Save
	invalidations <-chan events.Invalidation
}

func (e *postgresEvents) Saves() <-chan events.Save {
	return e.saves
}

func (e *postgresEvents) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}

func forwardNotifications(incoming <-chan *pq.Notification, saves chan<- events.Save, invalidations chan<- events.Invalidation) {
	// TODO: Implement this
}
