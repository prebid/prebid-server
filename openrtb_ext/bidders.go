package openrtb_ext

type BidderName string

const (
	BidderAppnexus BidderName = "appnexus"
)

func (name BidderName) MarshalJSON() ([]byte, error) {
	return []byte(name), nil
}

func (name *BidderName) String() string {
	if name == nil {
		return ""
	} else {
		return string(*name)
	}
}
