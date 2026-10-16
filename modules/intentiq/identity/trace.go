package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/enrichment"
)

const traceExtKey = "iiq-identity"

type flowTrace struct {
	collect bool
	msgs    []string
}

func (t *flowTrace) tracef(format string, args ...any) {
	if !t.enabled() {
		return
	}
	t.msgs = append(t.msgs, fmt.Sprintf(format, args...))
}

func (t *flowTrace) enabled() bool {
	return t != nil && t.collect
}

func requestTraceOptIn(req *openrtb2.BidRequest) bool {
	if req == nil || len(req.Ext) == 0 {
		return false
	}
	if b, err := jsonparser.GetBoolean(req.Ext, "prebid", "debug"); err == nil && b {
		return true
	}
	if s, err := jsonparser.GetString(req.Ext, "prebid", "trace"); err == nil && s != "" {
		return true
	}
	return false
}

func setIdentityTraceMutation(messages []string) func(hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
	return func(p hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
		if p.BidResponse == nil || len(messages) == 0 {
			return p, nil
		}
		b, err := json.Marshal(messages)
		if err != nil {
			return p, fmt.Errorf("intentiq-identity: marshal trace: %w", err)
		}
		ext := p.BidResponse.Ext
		if len(bytes.TrimSpace(ext)) == 0 {
			ext = []byte("{}")
		}
		ext, err = jsonparser.Set(ext, b, "trace", traceExtKey)
		if err != nil {
			return p, fmt.Errorf("intentiq-identity: set ext.trace.%s: %w", traceExtKey, err)
		}
		p.BidResponse.Ext = ext
		return p, nil
	}
}

func fmtDur(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return d.String()
	case d < time.Millisecond:
		return d.Round(time.Microsecond).String()
	default:
		return d.Round(time.Millisecond).String()
	}
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func presentAbsent(b bool) string {
	if b {
		return "present"
	}
	return "absent"
}

// Shows which candidate keys the request could produce.
func requestSignals(req *openrtb2.BidRequest) string {
	eids, ifa, ip, consent := 0, "none", "", false
	if req != nil {
		if req.User != nil {
			eids = len(req.User.EIDs)
			consent = req.User.Consent != ""
		}
		if d := req.Device; d != nil {
			if notBlank(d.IFA) {
				ifa = "present"
			}
			if ip = d.IP; ip == "" {
				ip = d.IPv6
			}
		}
	}
	return fmt.Sprintf("eids=%d, device.ifa=%s, ip=%s, consent=%s", eids, ifa, ip, presentAbsent(consent))
}

func keysDetail(keys []cache.Key) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s(%s)", k.Key, k.Type.Token()))
	}
	return strings.Join(parts, ", ")
}

func eidsDetail(eids []openrtb2.EID) string {
	parts := make([]string, 0, len(eids))
	for _, e := range eids {
		ids := make([]string, 0, len(e.UIDs))
		for _, u := range e.UIDs {
			ids = append(ids, u.ID)
		}
		parts = append(parts, fmt.Sprintf("%s[%s]", e.Source, strings.Join(ids, ",")))
	}
	return strings.Join(parts, ", ")
}

func tcStr(tc *int64) string {
	if tc == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *tc)
}

func cttlStr(r enrichment.Response) string {
	if r.Cttl == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *r.Cttl)
}

// Keeps head/tail + length; the IntentIQ abTestUuid is too long to read inline.
func abTestShort(s string) string {
	if s == "" {
		return "none"
	}
	if n := len(s); n > 24 {
		return fmt.Sprintf("%s…%s (%d chars)", s[:12], s[n-6:], n)
	}
	return s
}
