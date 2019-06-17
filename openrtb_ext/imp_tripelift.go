package openrtb_ext

import "encoding/json"

// ExtImpTriplelift defines the contract for bidrequest.imp[i].ext.triplelift
type ExtImpTriplelift struct {
	SSP                     string                  `json:"ssp"`
	PaymentChain            string                  `json:"pchain"`
	AppID                   string                  `json:"appId"`
	SupplyChain             *ExtTLSupplyChain       `json:"schain"`
    InvCode                 string                  `json:"inv_code"`
}

// ExtTLSupplyChainNode defines the format of bidrequest.imp[i].ext.triplelift.schain.nodes[n]
type ExtTLSupplyChainNode struct {
    ASI    string `json:"asi"`
    PID    string `json:"pid"`
    RID    string `json:"rid"`
    NAME   string `json:"name"`
    DOMAIN string `json:"domain"`
}

// ExtTLSupplyChain defines the format of bidrequest.imp[i].ext.triplelift.schain
type ExtTLSupplyChain struct {
	Complete    int   `json:"complete"`
	Values      []*ExtTLSupplyChainNode `json:"nodes"`
}
