package gdpr

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestSignalParse(t *testing.T) {
	tests := []struct {
		description string
		rawSignal   string
		wantSignal  Signal
		wantError   bool
	}{
		{
			description: "valid raw signal is 0",
			rawSignal:   "0",
			wantSignal:  SignalNo,
			wantError:   false,
		},
		{
			description: "Valid signal - raw signal is 1",
			rawSignal:   "1",
			wantSignal:  SignalYes,
			wantError:   false,
		},
		{
			description: "Valid signal - raw signal is empty",
			rawSignal:   "",
			wantSignal:  SignalAmbiguous,
			wantError:   false,
		},
		{
			description: "Invalid signal - raw signal is -1",
			rawSignal:   "-1",
			wantSignal:  SignalAmbiguous,
			wantError:   true,
		},
		{
			description: "Invalid signal - raw signal is abc",
			rawSignal:   "abc",
			wantSignal:  SignalAmbiguous,
			wantError:   true,
		},
	}

	for _, test := range tests {
		signal, err := SignalParse(test.rawSignal)

		assert.Equal(t, test.wantSignal, signal, test.description)

		if test.wantError {
			assert.NotNil(t, err, test.description)
		} else {
			assert.Nil(t, err, test.description)
		}
	}
}

func TestSignalNormalize(t *testing.T) {
	tests := []struct {
		description         string
		userSyncIfAmbiguous bool
		giveSignal          Signal
		wantSignal          Signal
	}{
		{
			description:         "Don't normalize - Signal No and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalNo,
			wantSignal:          SignalNo,
		},
		{
			description:         "Don't normalize - Signal No and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalNo,
			wantSignal:          SignalNo,
		},
		{
			description:         "Don't normalize - Signal Yes and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalYes,
			wantSignal:          SignalYes,
		},
		{
			description:         "Don't normalize - Signal Yes and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalYes,
			wantSignal:          SignalYes,
		},
		{
			description:         "Normalize - Signal Ambiguous and userSyncIfAmbiguous false",
			userSyncIfAmbiguous: false,
			giveSignal:          SignalAmbiguous,
			wantSignal:          SignalYes,
		},
		{
			description:         "Normalize - Signal Ambiguous and userSyncIfAmbiguous true",
			userSyncIfAmbiguous: true,
			giveSignal:          SignalAmbiguous,
			wantSignal:          SignalNo,
		},
	}

	for _, test := range tests {
		config := config.GDPR{
			UsersyncIfAmbiguous: test.userSyncIfAmbiguous,
		}

		normalizedSignal := SignalNormalize(test.giveSignal, config)

		assert.Equal(t, test.wantSignal, normalizedSignal, test.description)
	}
}
