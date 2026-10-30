package validation

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/ptrutil"
)

// Config is the host-level module configuration under hooks.modules.vastlint.validation.
// Account-level configuration may override Mode, RejectOn, Rules, SamplePercent,
// AnnotateBids and AnnotateRules.
type Config struct {
	// Mode is "monitor" (report only, on the bid path), "reject" (drop bids that
	// hit RejectOn) or "async" (validate after the hook returns; nothing on the
	// bid changes and the hook never waits for the engine).
	Mode string `json:"mode"`
	// RejectOn lists rule ids whose presence drops a bid in reject mode. Empty
	// means nothing is dropped even in reject mode. Ids are matched against
	// the engine catalog case-insensitively.
	RejectOn []string `json:"reject_on"`
	// Rules overrides rule levels: id to "error", "warning", "info" or "off".
	// Accepted as an object {"VAST-2.0-x": "off"} or, because YAML and
	// environment configuration lowercase keys and split them on dots, as a
	// list ["VAST-2.0-x=off"] or [{"id": "VAST-2.0-x", "level": "off"}].
	// Ids are matched against the engine catalog case-insensitively.
	Rules ruleLevels `json:"rules"`
	// MaxDocumentBytes skips larger creatives. Default 262144 (256 KiB).
	MaxDocumentBytes int `json:"max_document_bytes"`
	// SamplePercent checks this share of eligible bids, 1 to 100. Default 100.
	// Reject mode always checks every bid.
	SamplePercent int `json:"sample_percent"`
	// PoolSize sizes the engine's instance pool (wasm) or pinned worker threads
	// (native), and bounds the number of off-path workers. Default GOMAXPROCS.
	PoolSize int `json:"pool_size"`
	// AnnotateBids writes a summary to bid.ext.vastlint. Default true.
	AnnotateBids *bool `json:"annotate_bids"`
	// AnnotateRules includes the rule ids in bid.ext.vastlint. Default true;
	// false keeps the summary and leaves the list to the analytics tags.
	AnnotateRules *bool `json:"annotate_rules"`
	// AsyncQueue bounds how many creatives wait for off-path validation in
	// async mode. Beyond it creatives are counted as not validated. Default 4096.
	AsyncQueue int `json:"async_queue"`
	// AsyncQueueBytes bounds the markup held by the async queue, in bytes, so
	// a burst of large creatives cannot grow memory without limit. Default
	// 67108864 (64 MiB).
	AsyncQueueBytes int `json:"async_queue_bytes"`
	// CacheSize is the size in bytes of the in-process verdict cache, keyed by
	// a hash of the creative, the effective rules and the engine version.
	// 0 (default) runs without a cache; freecache rounds anything smaller up
	// to 512 KiB.
	CacheSize int `json:"cache_size"`
	// CacheTTLSeconds bounds how long a verdict stays cached. Default 600.
	CacheTTLSeconds int `json:"cache_ttl_seconds"`
	// ReportIntervalSeconds is how often the module writes its own counters
	// (off-path totals, cache hit rate) to the server log, when they changed.
	// Default 0 (off): the same figures are available from the analytics tags.
	ReportIntervalSeconds *int `json:"report_interval_seconds"`
}

const (
	modeMonitor = "monitor"
	modeReject  = "reject"
	modeAsync   = "async"

	// maxBytesSetting bounds the byte-sized settings at 1 GiB, which also
	// keeps them inside int on 32-bit platforms.
	maxBytesSetting = 1 << 30
)

func validMode(m string) bool { return m == modeMonitor || m == modeReject || m == modeAsync }

// ruleLevels is the rules override map. It unmarshals from three shapes: an
// object keyed by rule id (nested objects are joined back with dots, which is
// what a YAML loader that splits keys on dots produces), a list of
// "id=level" strings, or a list of {"id","level"} objects.
type ruleLevels map[string]string

func (r *ruleLevels) UnmarshalJSON(b []byte) error {
	out := ruleLevels{}
	trimmed := strings.TrimSpace(string(b))
	switch {
	case trimmed == "" || trimmed == "null":
	case strings.HasPrefix(trimmed, "{"):
		var m map[string]interface{}
		if err := jsonutil.UnmarshalValid(b, &m); err != nil {
			return err
		}
		if err := flattenRules("", m, out); err != nil {
			return err
		}
	case strings.HasPrefix(trimmed, "["):
		var items []interface{}
		if err := jsonutil.UnmarshalValid(b, &items); err != nil {
			return err
		}
		for _, it := range items {
			switch v := it.(type) {
			case string:
				id, level, ok := strings.Cut(v, "=")
				if !ok {
					id, level, ok = strings.Cut(v, ":")
				}
				if !ok {
					return fmt.Errorf("rules entry %q must be \"id=level\"", v)
				}
				out[strings.TrimSpace(id)] = strings.TrimSpace(level)
			case map[string]interface{}:
				id, _ := v["id"].(string)
				level, _ := v["level"].(string)
				if id == "" || level == "" {
					return fmt.Errorf("rules entry must have \"id\" and \"level\"")
				}
				out[id] = level
			default:
				return fmt.Errorf("rules entry must be a string or an object, got %T", it)
			}
		}
	default:
		return fmt.Errorf("rules must be an object or a list")
	}
	*r = out
	return nil
}

// flattenRules turns {"vast-2": {"0-x": "off"}} back into {"vast-2.0-x": "off"}.
func flattenRules(prefix string, m map[string]interface{}, out ruleLevels) error {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case string:
			out[key] = t
		case map[string]interface{}:
			if err := flattenRules(key, t, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("rules[%q] must be a level string, got %T", key, v)
		}
	}
	return nil
}

// catalog is the engine's rule set with a case-insensitive index, so ids
// that a configuration loader lowercased still resolve to the canonical id.
type catalog struct {
	ids   map[string]bool
	lower map[string]string
}

func newCatalog(ids []string) *catalog {
	c := &catalog{ids: make(map[string]bool, len(ids)), lower: make(map[string]string, len(ids))}
	for _, id := range ids {
		c.ids[id] = true
		c.lower[strings.ToLower(id)] = id
	}
	return c
}

// canonical returns the catalog's spelling of id, or false when unknown.
func (c *catalog) canonical(id string) (string, bool) {
	if c.ids[id] {
		return id, true
	}
	canon, ok := c.lower[strings.ToLower(id)]
	return canon, ok
}

// resolve rewrites rules and rejectOn to canonical ids and reports the first
// unknown id or invalid level.
func (c *catalog) resolve(rules ruleLevels, rejectOn []string) (ruleLevels, []string, error) {
	outRules := make(ruleLevels, len(rules))
	for id, level := range rules {
		canon, ok := c.canonical(id)
		if !ok {
			return nil, nil, fmt.Errorf("rules names unknown rule %q", id)
		}
		if !validLevel(level) {
			return nil, nil, fmt.Errorf("rules[%q] must be error, warning, info or off, got %q", id, level)
		}
		outRules[canon] = level
	}
	outReject := make([]string, 0, len(rejectOn))
	for _, id := range rejectOn {
		canon, ok := c.canonical(id)
		if !ok {
			return nil, nil, fmt.Errorf("reject_on names unknown rule %q", id)
		}
		outReject = append(outReject, canon)
	}
	sort.Strings(outReject)
	return outRules, outReject, nil
}

func validLevel(l string) bool { return l == "error" || l == "warning" || l == "info" || l == "off" }

func parseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := jsonutil.UnmarshalValid(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("vastlint: invalid module config: %w", err)
		}
	}
	if cfg.Mode == "" {
		cfg.Mode = modeMonitor
	}
	if !validMode(cfg.Mode) {
		return cfg, fmt.Errorf("vastlint: mode must be %q, %q or %q, got %q", modeMonitor, modeReject, modeAsync, cfg.Mode)
	}
	if cfg.MaxDocumentBytes == 0 {
		cfg.MaxDocumentBytes = 256 << 10
	}
	if cfg.MaxDocumentBytes < 1 || cfg.MaxDocumentBytes > 4<<20 {
		return cfg, fmt.Errorf("vastlint: max_document_bytes must be 1 to 4194304")
	}
	if cfg.SamplePercent == 0 {
		cfg.SamplePercent = 100
	}
	if cfg.SamplePercent < 1 || cfg.SamplePercent > 100 {
		return cfg, fmt.Errorf("vastlint: sample_percent must be 1 to 100")
	}
	if cfg.PoolSize < 0 || cfg.PoolSize > 1024 {
		return cfg, fmt.Errorf("vastlint: pool_size must be 0 (GOMAXPROCS) to 1024")
	}
	if cfg.AnnotateBids == nil {
		cfg.AnnotateBids = ptrutil.ToPtr(true)
	}
	if cfg.AnnotateRules == nil {
		cfg.AnnotateRules = ptrutil.ToPtr(true)
	}
	if cfg.AsyncQueue == 0 {
		cfg.AsyncQueue = 4096
	}
	if cfg.AsyncQueue < 1 || cfg.AsyncQueue > 1<<20 {
		return cfg, fmt.Errorf("vastlint: async_queue must be 1 to 1048576")
	}
	if cfg.AsyncQueueBytes == 0 {
		cfg.AsyncQueueBytes = 64 << 20
	}
	if cfg.AsyncQueueBytes < 1<<16 || cfg.AsyncQueueBytes > maxBytesSetting {
		return cfg, fmt.Errorf("vastlint: async_queue_bytes must be 65536 to %d", maxBytesSetting)
	}
	if cfg.AsyncQueueBytes < cfg.MaxDocumentBytes {
		return cfg, fmt.Errorf("vastlint: async_queue_bytes (%d) must be at least max_document_bytes (%d)", cfg.AsyncQueueBytes, cfg.MaxDocumentBytes)
	}
	if cfg.CacheSize < 0 || cfg.CacheSize > maxBytesSetting {
		return cfg, fmt.Errorf("vastlint: cache_size must be 0 to %d bytes", maxBytesSetting)
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = 600
	}
	if cfg.CacheTTLSeconds < 1 {
		return cfg, fmt.Errorf("vastlint: cache_ttl_seconds must be positive")
	}
	if cfg.ReportIntervalSeconds == nil {
		cfg.ReportIntervalSeconds = ptrutil.ToPtr(0)
	}
	if *cfg.ReportIntervalSeconds < 0 || *cfg.ReportIntervalSeconds > 3600 {
		return cfg, fmt.Errorf("vastlint: report_interval_seconds must be 0 (off) to 3600")
	}
	for id, level := range cfg.Rules {
		if !validLevel(level) {
			return cfg, fmt.Errorf("vastlint: rules[%q] must be error, warning, info or off, got %q", id, level)
		}
	}
	sort.Strings(cfg.RejectOn)
	return cfg, nil
}

// accountOverrides is the subset of Config an account may change.
type accountOverrides struct {
	Mode          *string    `json:"mode"`
	RejectOn      *[]string  `json:"reject_on"`
	Rules         ruleLevels `json:"rules"`
	SamplePercent *int       `json:"sample_percent"`
	AnnotateBids  *bool      `json:"annotate_bids"`
	AnnotateRules *bool      `json:"annotate_rules"`
}

// effective returns the host config with account overrides applied. Account
// rules are merged over host rules per rule id; the other fields replace the
// host value. cat is the engine's rule catalog; an override that names a rule
// outside it, or any other invalid account config, is reported and the host
// config is used unchanged, so one account's typo costs it the override and
// nothing else.
func (c Config) effective(account json.RawMessage, cat *catalog) (Config, error) {
	if len(account) == 0 {
		return c, nil
	}
	var o accountOverrides
	if err := jsonutil.UnmarshalValid(account, &o); err != nil {
		return c, fmt.Errorf("vastlint: invalid account config: %w", err)
	}
	out := c
	if o.Mode != nil {
		if !validMode(*o.Mode) {
			return c, fmt.Errorf("vastlint: account mode must be %q, %q or %q", modeMonitor, modeReject, modeAsync)
		}
		out.Mode = *o.Mode
	}
	if o.RejectOn != nil || len(o.Rules) > 0 {
		accountRules, accountReject := o.Rules, out.RejectOn
		if o.RejectOn != nil {
			accountReject = *o.RejectOn
		}
		if cat != nil {
			var err error
			if accountRules, accountReject, err = cat.resolve(accountRules, accountReject); err != nil {
				return c, fmt.Errorf("vastlint: account config: %w", err)
			}
		} else {
			accountReject = append([]string(nil), accountReject...)
			sort.Strings(accountReject)
		}
		if o.RejectOn != nil {
			out.RejectOn = accountReject
		}
		if len(accountRules) > 0 {
			merged := make(ruleLevels, len(c.Rules)+len(accountRules))
			for id, level := range c.Rules {
				merged[id] = level
			}
			for id, level := range accountRules {
				merged[id] = level
			}
			out.Rules = merged
		}
	}
	if o.SamplePercent != nil {
		if *o.SamplePercent < 1 || *o.SamplePercent > 100 {
			return c, fmt.Errorf("vastlint: account sample_percent must be 1 to 100")
		}
		out.SamplePercent = *o.SamplePercent
	}
	if o.AnnotateBids != nil {
		out.AnnotateBids = o.AnnotateBids
	}
	if o.AnnotateRules != nil {
		out.AnnotateRules = o.AnnotateRules
	}
	return out, nil
}

func (c Config) rejects(ruleID string) bool {
	for _, id := range c.RejectOn {
		if id == ruleID {
			return true
		}
	}
	return false
}
