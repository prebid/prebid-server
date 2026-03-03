package gdpr

import (
	"strconv"

	"github.com/prebid/prebid-server/v3/errortypes"
)

type Signal int

const (
	SignalAmbiguous Signal = -1
	SignalNo        Signal = 0
	SignalYes       Signal = 1
)

var gdprSignalError = &errortypes.BadInput{Message: "GDPR signal should be integer 0 or 1"}

// StrSignalParse returns a parsed GDPR signal or a parse error.
func StrSignalParse(signal string) (Signal, error) {
	if signal == "" {
		return SignalAmbiguous, nil
	}

	i, err := strconv.Atoi(signal)

	if err != nil {
		return SignalAmbiguous, gdprSignalError
	}

	return IntSignalParse(i)
}

// IntSignalParse checks parameter i is not out of bounds and returns a GDPR signal error
func IntSignalParse(i int) (Signal, error) {
	if i != 0 && i != 1 {
		return SignalAmbiguous, gdprSignalError
	}

	return Signal(i), nil
}

// SignalNormalize normalizes a GDPR signal to ensure it's always either SignalYes or SignalNo.
func SignalNormalize(signal Signal, gdprDefaultValue string) Signal {
	if signal != SignalAmbiguous {
		return signal
	}

	if gdprDefaultValue == "0" {
		return SignalNo
	}

	return SignalYes
}
