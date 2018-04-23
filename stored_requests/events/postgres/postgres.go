package postgres

import (
	"encoding/json"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
)

// The Postgres EventProducer uses Listen/Notify to pick up changes from the database.
// The channel names are configurable, but the message payloads must conform to the following formats:
//
// A "delete" notification should contain an array of IDs, like this:
//
//   ["id-1", "id-2", "id-3"]
//
// An "update" or "insert" notification should contain a map from ID to data, like this:
//
// {
//   "id-1": { ... Stored Data for id-1 ... },
//   "id-2": { ... Stored Data for id-2 ...}
// }
//
// These events will used to update or invalidate data in the OpenRTB or AMP caches depending on
// the Channel where this message was heard.
//
// For more info on Listen/Notify, see:
//
//   - https://www.postgresql.org/docs/current/static/sql-listen.html
//   - https://www.postgresql.org/docs/current/static/sql-notify.html
//   - https://www.postgresql.org/docs/current/static/plpgsql-trigger.html
func NewEvents(cfg *config.PostgresEventsConfig) (eventProducer events.EventProducer, ampEventProducer events.EventProducer, shutdown func()) {
	listener := pq.NewListener(cfg.ConnectionInfo.ConnString(), time.Duration(cfg.MinReconnectInterval)*time.Millisecond, time.Duration(cfg.MaxReconnectInterval)*time.Millisecond, nil)
	doListen(listener, cfg.Channels.OpenRTBRequestUpdates, "OpenRTB Stored Request Updates")
	doListen(listener, cfg.Channels.OpenRTBRequestDeletes, "OpenRTB Stored Request Deletes")

	doListen(listener, cfg.Channels.OpenRTBImpUpdates, "OpenRTB Stored Imp Updates")
	doListen(listener, cfg.Channels.OpenRTBImpDeletes, "OpenRTB Stored Imp Deletes")

	doListen(listener, cfg.Channels.AMPRequestUpdates, "AMP Stored Request Updates")
	doListen(listener, cfg.Channels.AMPRequestDeletes, "AMP Stored Request Deletes")

	openrtbEvents := &postgresEvents{
		saves:         make(chan events.Save, 10),
		invalidations: make(chan events.Invalidation, 10),
	}

	ampEvents := &postgresEvents{
		saves:         make(chan events.Save, 10),
		invalidations: make(chan events.Invalidation, 10),
	}

	go forwardNotifications(&cfg.Channels, listener.NotificationChannel(), openrtbEvents, ampEvents)

	return openrtbEvents, ampEvents, func() {
		if err := listener.Close(); err != nil {
			glog.Errorf("Error closing Postgres EventProducers: %v", err)
		}
	}
}

func doListen(listener *pq.Listener, channel string, updateType string) {
	glog.Infof("Listening for %s in Postgres on channel %s", updateType, channel)
	if err := listener.Listen(channel); err != nil {
		glog.Fatalf("Postgres notifier falied to listen on channel %s: %v", channel, err)
	}
}

func forwardNotifications(channels *config.PostgresEventsChannels, incoming <-chan *pq.Notification, openrtbEvents *postgresEvents, ampEvents *postgresEvents) {
	for {
		notification := <-incoming
		switch notification.Channel {
		case channels.OpenRTBRequestUpdates:
			openrtbEvents.saves <- events.Save{
				Requests: parseUpdateData(notification.Extra, channels.OpenRTBRequestUpdates),
			}
		case channels.OpenRTBRequestDeletes:
			openrtbEvents.invalidations <- events.Invalidation{
				Requests: parseDeleteData(notification.Extra, channels.OpenRTBRequestDeletes),
			}
		case channels.OpenRTBImpUpdates:
			openrtbEvents.saves <- events.Save{
				Imps: parseUpdateData(notification.Extra, channels.OpenRTBImpUpdates),
			}
		case channels.OpenRTBImpDeletes:
			openrtbEvents.invalidations <- events.Invalidation{
				Imps: parseDeleteData(notification.Extra, channels.OpenRTBImpDeletes),
			}
		case channels.AMPRequestUpdates:
			ampEvents.saves <- events.Save{
				Requests: parseUpdateData(notification.Extra, channels.AMPRequestUpdates),
			}
		case channels.AMPRequestDeletes:
			ampEvents.invalidations <- events.Invalidation{
				Requests: parseDeleteData(notification.Extra, channels.AMPRequestDeletes),
			}
		default:
			glog.Errorf("Postgres EventProducer received message on unknown channel: %s. Ignoring message", notification.Channel)
		}
	}
}

func parseUpdateData(msg string, channel string) (parsed map[string]json.RawMessage) {
	if err := json.Unmarshal(json.RawMessage(msg), &parsed); err != nil {
		glog.Errorf("Bad message on channel %s. Message was %s. Error was %v", channel, msg, err)
	}
	return
}

func parseDeleteData(msg string, channel string) (parsed []string) {
	if err := json.Unmarshal(json.RawMessage(msg), &parsed); err != nil {
		glog.Errorf("Bad message on channel %s. Message was %s. Error was %v", channel, msg, err)
	}
	return
}

type postgresEvents struct {
	saves         chan events.Save
	invalidations chan events.Invalidation
}

func (e *postgresEvents) Saves() <-chan events.Save {
	return e.saves
}

func (e *postgresEvents) Invalidations() <-chan events.Invalidation {
	return e.invalidations
}
