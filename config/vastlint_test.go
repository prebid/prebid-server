package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestVastlintEnvironmentConfig(t *testing.T) {
	for _, fromFile := range []bool{false, true} {
		name := "environment_only"
		if fromFile {
			name = "environment_overrides_file"
		}
		t.Run(name, func(t *testing.T) {
			settings := map[string]string{
				"ENABLED": "true", "ENDPOINT": "localhost:50051", "ALLOW_INSECURE": "true",
				"ACCOUNTS": "host-account second-account", "SAMPLE_PERCENT": "17", "MAX_DOCUMENT_BYTES": "32768",
				"QUEUE_SIZE": "12", "TIMEOUT_MS": "123", "MAX_AGE_MS": "777",
				"MAX_EVENTS_PER_SECOND": "11", "MAX_BYTES_PER_SECOND": "65536",
			}
			for key, value := range settings {
				t.Setenv("PBS_ANALYTICS_VASTLINT_"+key, value)
			}
			v := viper.New()
			SetupViper(v, "", bidderInfos)
			v.Set("gdpr.default_value", "0")
			if fromFile {
				v.SetConfigType("yaml")
				require.NoError(t, v.ReadConfig(strings.NewReader(`analytics:
  vastlint:
    enabled: false
    endpoint: old-host:443
    accounts: [old-account]
    sample_percent: 1
    queue_size: 64
`)))
			}
			cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
			require.NoError(t, err)
			want := VastlintAnalytics{Enabled: true, Endpoint: "localhost:50051", AllowInsecure: true,
				Accounts: []string{"host-account", "second-account"}, SamplePercent: 17, MaxDocumentBytes: 32768,
				QueueSize: 12, TimeoutMS: 123, MaxAgeMS: 777, MaxEventsPerSecond: 11, MaxBytesPerSecond: 65536}
			require.Equal(t, want, cfg.Analytics.Vastlint)
		})
	}
}
