package openrtb_ext

type BidderName string

const (
	BidderAppnexus BidderName = "appnexus"
)

var bidderMap = map[string]BidderName {
	"appnexus": BidderAppnexus,
}

func (name BidderName) MarshalJSON() ([]byte, error) {
	return []byte(name.String()), nil
}

func (name *BidderName) String() string {
	if name == nil {
		return ""
	} else {
		return string(*name)
	}
}
