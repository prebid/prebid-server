package main

import (
	"os"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"
)

func compareStrings(t *testing.T, message string, expect string, actual string) {
	if expect != actual {
		t.Errorf(message, expect, actual)
	}
}

// forceEnv sets an environment variable to a certain value, and return a deferable function to reset it to the original value.
func forceEnv(t *testing.T, key string, val string) func() {
	orig, set := os.LookupEnv(key)
	err := os.Setenv(key, val)
	if err != nil {
		t.Fatalf("Error setting environment %s", key)
	}
	if set {
		return func() {
			if os.Setenv(key, orig) != nil {
				t.Fatalf("Error unsetting environment %s", key)
			}
		}
	}
	return func() {
		if os.Unsetenv(key) != nil {
			t.Fatalf("Error unsetting environment %s", key)
		}
	}
}

// Test the viper setup
func TestViperInit(t *testing.T) {
	v := viper.New()
	config.SetupViper(v, "", nil)
	compareStrings(t, "Viper error: external_url expected to be %s, found %s", "http://localhost:8000", v.Get("external_url").(string))
	compareStrings(t, "Viper error: accounts.filesystem.directorypath expected to be %s, found %s", "./stored_requests/data/by_id", v.Get("accounts.filesystem.directorypath").(string))
}

func TestViperEnv(t *testing.T) {
	v := viper.New()
	config.SetupViper(v, "", nil)
	port := forceEnv(t, "PBS_PORT", "7777")
	defer port()

	endpt := forceEnv(t, "PBS_EXPERIMENT_ADSCERT_INPROCESS_ORIGIN", "not_an_endpoint")
	defer endpt()

	ttl := forceEnv(t, "PBS_HOST_COOKIE_TTL_DAYS", "60")
	defer ttl()

	ipv4Networks := forceEnv(t, "PBS_REQUEST_VALIDATION_IPV4_PRIVATE_NETWORKS", "1.1.1.1/24 2.2.2.2/24")
	defer ipv4Networks()

	assert.Equal(t, 7777, v.Get("port"), "Basic Config")
	assert.Equal(t, "not_an_endpoint", v.Get("experiment.adscert.inprocess.origin"), "Nested Config")
	assert.Equal(t, 60, v.Get("host_cookie.ttl_days"), "Config With Underscores")
	assert.ElementsMatch(t, []string{"1.1.1.1/24", "2.2.2.2/24"}, v.Get("request_validation.ipv4_private_networks"), "Arrays")
}
