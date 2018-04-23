package postgres

import (
	"encoding/json"
	"testing"

	"github.com/lib/pq"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/events"
)

func TestOpenRTBRequestUpdate(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		OpenRTBRequestUpdates: "req-updates",
	}
	ortb, _ := runTest(cfg, "req-updates", `{"req-1":true}`)
	gotSave := <-ortb.Saves()
	assertMapLength(t, gotSave.Requests, 1)
	assertMapValue(t, gotSave.Requests, "req-1", "true")
	assertMapLength(t, gotSave.Imps, 0)
}

func TestOpenRTBRequestDelete(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		OpenRTBRequestDeletes: "req-deletes",
	}
	ortb, _ := runTest(cfg, "req-deletes", `["req-1"]`)
	gotInvalidation := <-ortb.Invalidations()
	assertSliceLength(t, gotInvalidation.Requests, 1)
	assertSliceContains(t, gotInvalidation.Requests, "req-1")
	assertSliceLength(t, gotInvalidation.Imps, 0)
}

func TestOpenRTBImpUpdate(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		OpenRTBImpUpdates: "imp-updates",
	}
	ortb, _ := runTest(cfg, "imp-updates", `{"imp-1":true}`)
	gotSave := <-ortb.Saves()
	assertMapLength(t, gotSave.Imps, 1)
	assertMapValue(t, gotSave.Imps, "imp-1", "true")
	assertMapLength(t, gotSave.Requests, 0)
}

func TestOpenRTBImpDelete(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		OpenRTBImpDeletes: "imp-deletes",
	}
	ortb, _ := runTest(cfg, "imp-deletes", `["imp-1"]`)
	gotInvalidation := <-ortb.Invalidations()
	assertSliceLength(t, gotInvalidation.Imps, 1)
	assertSliceContains(t, gotInvalidation.Imps, "imp-1")
	assertSliceLength(t, gotInvalidation.Requests, 0)
}

func TestAMPRequestUpdate(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		AMPRequestUpdates: "amp-updates",
	}
	_, amp := runTest(cfg, "amp-updates", `{"req-1":true}`)
	gotSave := <-amp.Saves()
	assertMapLength(t, gotSave.Requests, 1)
	assertMapValue(t, gotSave.Requests, "req-1", "true")
	assertMapLength(t, gotSave.Imps, 0)
}

func TestAMPRequestDelete(t *testing.T) {
	cfg := &config.PostgresEventsChannels{
		AMPRequestDeletes: "req-deletes",
	}
	_, amp := runTest(cfg, "req-deletes", `["req-1"]`)
	gotInvalidation := <-amp.Invalidations()
	assertSliceLength(t, gotInvalidation.Requests, 1)
	assertSliceContains(t, gotInvalidation.Requests, "req-1")
	assertSliceLength(t, gotInvalidation.Imps, 0)
}

func runTest(cfg *config.PostgresEventsChannels, channel string, payload string) (ortb *postgresEvents, amp *postgresEvents) {
	ortb = &postgresEvents{
		saves:         make(chan events.Save),
		invalidations: make(chan events.Invalidation),
	}
	amp = &postgresEvents{
		saves:         make(chan events.Save),
		invalidations: make(chan events.Invalidation),
	}
	incoming := make(chan *pq.Notification)
	go forwardNotifications(cfg, incoming, ortb, amp)
	incoming <- &pq.Notification{
		Channel: channel,
		Extra:   payload,
	}
	return
}

func assertMapLength(t *testing.T, m map[string]json.RawMessage, expected int) {
	t.Helper()
	if len(m) != expected {
		t.Errorf("Bad map length. Expected %d, got %d", expected, len(m))
	}
}

func assertSliceLength(t *testing.T, data []string, expected int) {
	t.Helper()
	if len(data) != expected {
		t.Errorf("Bads slice length. Expected %d, got %d", expected, len(data))
	}
}

func assertSliceContains(t *testing.T, data []string, value string) {
	t.Helper()
	for _, val := range data {
		if val == value {
			return
		}
	}
	t.Errorf("Slice %v didn't contain expected value %s", data, value)
}

func assertMapValue(t *testing.T, m map[string]json.RawMessage, key string, expectedValue string) {
	t.Helper()
	if val := m[key]; string(val) != expectedValue {
		t.Errorf("Bad map[%s]. Expected %s, got %s", key, expectedValue, string(val))
	}
}
