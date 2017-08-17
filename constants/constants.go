package constants

// Family Name type to verify and centralize FamilyName usage
type FamilyName int

const (
	FNAppnexus FamilyName = iota
	FNFacebook
	FNIndex
	FNPubmatic
	FNPulsepoint
	FNRubicon
)

var validFamilyNames = [...]string{
	"adnxs",
	"audienceNetwork",
	"indexExchange",
	"pubmatic",
	"pulsepoint",
	"rubicon",
}

func (n FamilyName) String() string {
	return validFamilyNames[int(n)]
}
