package openrtb_ext

// ExtImpRoundhouseads defines the contract for bidrequest.imp[i].ext
// PublisherId is a mandatory parameter for the Roundhouseads bidder.

type ExtImpRoundhouseads struct {
	PublisherId string `json:"publisherId"`
}
