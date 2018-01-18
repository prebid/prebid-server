package exchange

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestAdapterList(t *testing.T) {
	list := AdapterList()
	adapterInList(t, openrtb_ext.BidderAppnexus, list)
	adapterInList(t, openrtb_ext.BidderRubicon, list)
	adapterInList(t, openrtb_ext.BidderPubmatic, list)
	adapterInList(t, openrtb_ext.BidderFacebook, list)
	adapterInList(t, openrtb_ext.BidderIndex, list)
	adapterInList(t, openrtb_ext.BidderPulsepoint, list)
	adapterInList(t, openrtb_ext.BidderConversant, list)
}

func adapterInList(t *testing.T, a openrtb_ext.BidderName, l []openrtb_ext.BidderName) {
	found := false
	for _, n := range l {
		if a == n {
			found = true
		}
	}
	if !found {
		t.Errorf("Adapter %s not found in the adapter map!", a)
	}
}
