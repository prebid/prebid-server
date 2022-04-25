package gdpr

import (
	"strconv"

	"github.com/prebid/prebid-server/errortypes"
)

type Signal int

const (
	SignalAmbiguous Signal = -1
	SignalNo        Signal = 0
	SignalYes       Signal = 1
)

var gdprSignalError = &errortypes.BadInput{Message: "GDPR signal should be integer 0 or 1"}

// SignalParse returns a parsed GDPR signal or a parse error.
func SignalParse(rawSignal string) (Signal, error) {
	if rawSignal == "" {
		return SignalAmbiguous, nil
	}

	i, err := strconv.Atoi(rawSignal)

	if err != nil || (i != 0 && i != 1) {
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
