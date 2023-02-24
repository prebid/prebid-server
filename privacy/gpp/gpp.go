package gpp

// Policy represents the GPP privacy string container.
// Currently just a placeholder until more expansive support is made.
type Policy struct {
	Consent string
	RawSID  string // This is the CSV format ("2,6") that the IAB recommends for passing the SID(s) on a query string.
}
