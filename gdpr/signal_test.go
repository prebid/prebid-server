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
		{
			description: "Out of bounds signal - raw signal is 5",
			rawSignal:   "5",
			wantSignal:  SignalAmbiguous,
			wantError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			signal, err := StrSignalParse(test.rawSignal)

			assert.Equal(t, test.wantSignal, signal, test.description)

			if test.wantError {
				assert.NotNil(t, err, test.description)
			} else {
				assert.Nil(t, err, test.description)
			}
		})
	}
}

func TestIntSignalParse(t *testing.T) {
	type testOutput struct {
		signal Signal
		err    error
	}
	testCases := []struct {
		desc     string
		input    int
		expected testOutput
	}{
		{
			desc:  "input out of bounds, return SgnalAmbituous and gdprSignalError",
			input: -1,
			expected: testOutput{
				signal: SignalAmbiguous,
				err:    gdprSignalError,
			},
		},
		{
			desc:  "input in bounds equals signalNo, return signalNo and nil error",
			input: 0,
			expected: testOutput{
				signal: SignalNo,
				err:    nil,
			},
		},
		{
			desc:  "input in bounds equals signalYes, return signalYes and nil error",
			input: 1,
			expected: testOutput{
				signal: SignalYes,
				err:    nil,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			outSignal, outErr := IntSignalParse(tc.input)

			assert.Equal(t, tc.expected.signal, outSignal, tc.desc)
			assert.Equal(t, tc.expected.err, outErr, tc.desc)
		})
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
