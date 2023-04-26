package gdpr

import (
	"testing"

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
		description  string
		defaultValue string
		giveSignal   Signal
		wantSignal   Signal
	}{
		{
			description:  "Don't normalize - Signal No and Default Value 1",
			defaultValue: "1",
			giveSignal:   SignalNo,
			wantSignal:   SignalNo,
		},
		{
			description:  "Don't normalize - Signal No and Default Value 0",
			defaultValue: "0",
			giveSignal:   SignalNo,
			wantSignal:   SignalNo,
		},
		{
			description:  "Don't normalize - Signal Yes and Default Value 1",
			defaultValue: "1",
			giveSignal:   SignalYes,
			wantSignal:   SignalYes,
		},
		{
			description:  "Don't normalize - Signal Yes and Default Value 0",
			defaultValue: "0",
			giveSignal:   SignalYes,
			wantSignal:   SignalYes,
		},
		{
			description:  "Normalize - Signal Ambiguous and Default Value 1",
			defaultValue: "1",
			giveSignal:   SignalAmbiguous,
			wantSignal:   SignalYes,
		},
		{
			description:  "Normalize - Signal Ambiguous and Default Value 0",
			defaultValue: "0",
			giveSignal:   SignalAmbiguous,
			wantSignal:   SignalNo,
		},
	}

	for _, test := range tests {
		normalizedSignal := SignalNormalize(test.giveSignal, test.defaultValue)

		assert.Equal(t, test.wantSignal, normalizedSignal, test.description)
	}
}
