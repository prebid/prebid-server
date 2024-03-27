package dchain

import (
	"encoding/json"
	"strconv"

	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type Dchain struct {
	Ver      string          `json:"ver,omitempty"`
	Complete int             `json:"complete,omitempty"`
	Nodes    []DchainNodes   `json:"nodes,omitempty"`
	Ext      json.RawMessage `json:"ext,omitempty"`
}

type DchainNodes struct {
	Asi    string          `json:"asi,omitempty"`
	Bsid   string          `json:"bsid,omitempty"`
	Rid    string          `json:"rid,omitempty"`
	Name   string          `json:"name,omitempty"`
	Domain string          `json:"domain,omitempty"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

func IsValidDchain(dchain Dchain) bool {
	if dchain.Complete != 0 && dchain.Complete != 1 {
		return false
	}
	if dchain.Nodes == nil || len(dchain.Nodes) == 0 {
		return false
	}
	return true
}

func AddDchainNode(prebidMeta *openrtb_ext.ExtBidPrebidMeta) {
	var dchain Dchain
	if err := json.Unmarshal(prebidMeta.DChain, &dchain); err == nil && IsValidDchain(dchain) {
		dchain.Nodes = append(dchain.Nodes,
			DchainNodes{Asi: prebidMeta.AdapterCode},
		)
		prebidMeta.DChain, _ = json.Marshal(dchain)
		return
	}
	basicDchain := Dchain{
		Ver:      "1.0",
		Complete: 0,
		Nodes:    []DchainNodes{},
	}

	if prebidMeta.NetworkID != 0 && prebidMeta.NetworkName != "" {
		basicDchain.Nodes = append(basicDchain.Nodes,
			DchainNodes{Name: prebidMeta.NetworkName, Bsid: strconv.Itoa(prebidMeta.NetworkID)},
		)
	}

	basicDchain.Nodes = append(basicDchain.Nodes, DchainNodes{Name: prebidMeta.AdapterCode})
	prebidMeta.DChain, _ = json.Marshal(basicDchain)
}
