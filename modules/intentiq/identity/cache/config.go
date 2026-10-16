package cache

import "time"

type Config struct {
	Enabled                     bool   `json:"enabled"`
	Provider                    string `json:"provider"`
	TTLSeconds                  int    `json:"ttl_seconds"`
	MaxKeys                     int    `json:"max_keys"`
	MaxSize                     int    `json:"max_size"` // in-process (L1) byte budget
	TTLCeilingFirstPartySeconds int    `json:"ttl_ceiling_first_party_seconds"`
	TTLCeilingThirdPartySeconds int    `json:"ttl_ceiling_third_party_seconds"`
	TTLCeilingDeviceSeconds     int    `json:"ttl_ceiling_device_seconds"`
	NegativeTTLSeconds          int    `json:"negative_ttl_seconds"`
	InProgressTTLSeconds        int    `json:"in_progress_ttl_seconds"`
}

func (c Config) TTLPolicy() TTLPolicy {
	sec := func(s int) time.Duration { return time.Duration(s) * time.Second }
	return TTLPolicy{
		Default:           sec(c.TTLSeconds),
		FirstPartyCeiling: sec(c.TTLCeilingFirstPartySeconds),
		ThirdPartyCeiling: sec(c.TTLCeilingThirdPartySeconds),
		DeviceCeiling:     sec(c.TTLCeilingDeviceSeconds),
		NegativeTTL:       sec(c.NegativeTTLSeconds),
		InProgressTTL:     sec(c.InProgressTTLSeconds),
	}
}
