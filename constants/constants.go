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
var maxFamilyName = len(validFamilyNames)

func (n FamilyName) String() string {
	return validFamilyNames[int(n)]
}

// Define an array of UIDs, and functions to conver between int indexed array of UIDs, and
// the string indexed map of UIDs needed for PBSCookie

func NewUIDArray() []string {
	return make([]string, maxFamilyName)
}

// UIDsToArray(): Convert a map of UIDs into an array indexed with FamilyName
// we will toss any unknown FamilyNames.
func UIDsToArray(m map[string]string) []string {
	a := NewUIDArray()
	for i, v := range validFamilyNames {
		uid, ok := m[v]
		if ok {
			a[i] = uid
		}
	}
	return a
}

// UIDsToMap(): Convert an array of UIDs into a string indexed map of UIDs, as needed by PBSCookie
func UIDsToMap(a []string) map[string]string {
	m := make(map[string]string)
	for i, v := range validFamilyNames {
		if len(a[i]) > 0 {
			m[v] = a[i]
		}
	}
	return m
}
