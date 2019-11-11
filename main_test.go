package main

import (
	"os"
	"testing"

	"github.com/prebid/prebid-server/config"

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
	config.SetupViper(v, "")
	compareStrings(t, "Viper error: external_url expected to be %s, found %s", "http://localhost:8000", v.Get("external_url").(string))
	compareStrings(t, "Viper error: adapters.pulsepoint.endpoint expected to be %s, found %s", "http://bid.contextweb.com/header/s/ortb/prebid-s2s", v.Get("adapters.pulsepoint.endpoint").(string))
}

func TestViperEnv(t *testing.T) {
	v := viper.New()
	config.SetupViper(v, "")
	port := forceEnv(t, "PBS_PORT", "7777")
	defer port()

	endpt := forceEnv(t, "PBS_ADAPTERS_PUBMATIC_ENDPOINT", "not_an_endpoint")
	defer endpt()

	ttl := forceEnv(t, "PBS_HOST_COOKIE_TTL_DAYS", "60")
	defer ttl()

	// Basic config set
	compareStrings(t, "Viper error: port expected to be %s, found %s", "7777", v.Get("port").(string))
	// Nested config set
	compareStrings(t, "Viper error: adapters.pubmatic.endpoint expected to be %s, found %s", "not_an_endpoint", v.Get("adapters.pubmatic.endpoint").(string))
	// Config set with underscores
	compareStrings(t, "Viper error: host_cookie.ttl_days expected to be %s, found %s", "60", v.Get("host_cookie.ttl_days").(string))
}
