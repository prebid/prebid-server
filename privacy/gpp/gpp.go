package gpp

import (
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
)

// Policy represents the GPP privacy string container.
// Currently just a placeholder until more expansive support is made.
type Policy struct {
	Consent string
	RawSID  string // This is the CSV format ("2,6") that the IAB recommends for passing the SID(s) on a query string.
}

func IsSIDInList(gppSIDs []int8, sid gppConstants.SectionID) bool {
	var rc bool = false

	for _, id := range gppSIDs {
		if id == int8(sid) {
			rc = true
			break
		}
	}
	return rc
}

func IndexOfSID(gpp gpplib.GppContainer, sid gppConstants.SectionID) int {
	var rv int = -1

	for i, id := range gpp.SectionTypes {
		if id == sid {
			return i
		}
	}
	return rv
}
