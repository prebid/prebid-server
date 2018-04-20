package postgres

import (
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
)

func NewEvents(cfg *config.PostgresEventsConfig) (eventProducer events.EventProducer, ampEventProducer events.EventProducer, shutdown func() error) {
	listener := pq.NewListener(cfg.ConnectionInfo.ConnString(), time.Duration(cfg.MinReconnectInterval)*time.Millisecond, time.Duration(cfg.MaxReconnectInterval)*time.Millisecond, nil)
	doListen(listener, cfg.ORTBChannel, "Stored Request")
	doListen(listener, cfg.AMPChannel, "AMP Stored Request")

	saves := make(chan events.Save, 10)
	invalidations := make(chan events.Invalidation, 10)

	go forwardNotifications(listener.NotificationChannel(), saves, invalidations)

	return &postgresEvents{
			saves:         saves,
			invalidations: invalidations,
		}, &postgresEvents{
			saves:         saves,
			invalidations: invalidations,
		}, listener.Close
}

func doListen(listener *pq.Listener, channel string, updateType string) {
	glog.Infof("Listening for %s updates in Postgres on channel %s", updateType, channel)
	if err := listener.Listen(channel); err != nil {
		glog.Fatalf("Postgres notifier falied to listen on channel %s: %v", channel, err)
	}
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
	for {
		notification := <-incoming
		glog.Infof("Got notification: %s", notification.Extra)
		// TODO: Implement this for real
	}
}
