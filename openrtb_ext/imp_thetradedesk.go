package openrtb_ext

// ExtImpTheTradeDesk defines the contract for bidrequest.imp[i].ext
// PublisherId is mandatory parameters, others are optional parameters

type ExtImpTheTradeDesk struct {
	PublisherId string `json:"publisherId"`
}
