package family

// Family Name type to verify and centralize FamilyName usage
type Name int

const (
	Appnexus Name = iota
	Facebook
	Index
	Pubmatic
	Pulsepoint
	Rubicon
)

var validFamilyNames = [...]string{
	"adnxs",
	"audienceNetwork",
	"indexExchange",
	"pubmatic",
	"pulsepoint",
	"rubicon",
}

func (n Name) String() string {
	return validFamilyNames[int(n)]
}
