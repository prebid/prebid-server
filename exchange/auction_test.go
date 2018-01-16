package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestImpCount(t *testing.T) {
	a := newAuction(2)
	a.addBid(openrtb_ext.BidderAppnexus, &openrtb.Bid{
		ImpID: "imp-1",
	})
	a.addBid(openrtb_ext.BidderRubicon, &openrtb.Bid{
		ImpID: "imp-1",
	})
	a.addBid(openrtb_ext.BidderIndex, &openrtb.Bid{
		ImpID: "imp-2",
	})
	if len(a.winningBids) != 2 {
		t.Errorf("Expected 2 imps. Got %d", len(a.winningBids))
	}
}

func TestAuctionIntegrity(t *testing.T) {
	a := newAuction(2)
	oneImpId := "imp-1"
	otherImpId := "imp-2"

	apnWinner := &openrtb.Bid{
		ImpID: oneImpId,
		Price: 3,
	}
	apnLoser := &openrtb.Bid{
		ImpID: oneImpId,
		Price: 2,
	}
	apnCompetitor := &openrtb.Bid{
		ImpID: otherImpId,
		Price: 1,
	}
	rubiWinner := &openrtb.Bid{
		ImpID: otherImpId,
		Price: 2,
	}
	a.addBid(openrtb_ext.BidderAppnexus, apnWinner)
	a.addBid(openrtb_ext.BidderAppnexus, apnLoser)
	a.addBid(openrtb_ext.BidderRubicon, rubiWinner)
	a.addBid(openrtb_ext.BidderAppnexus, apnCompetitor)

	seenWinnerImp1 := false
	seenWinnerImp2 := false
	seenLoserImp1 := false
	seenLoserImp2 := false

	numBestBids := 0
	a.forEachBestBid(func(impId string, bidderName openrtb_ext.BidderName, bid *openrtb.Bid, winner bool) {
		numBestBids++

		if bid == apnWinner {
			seenWinnerImp1 = true
		}
		if bid == apnLoser {
			seenLoserImp1 = true
		}
		if bid == rubiWinner {
			seenWinnerImp2 = true
		}
		if bid == apnCompetitor {
			seenLoserImp2 = true
		}
	})

	if !seenWinnerImp1 {
		t.Errorf("foreachBestBid did not execute on apn winning bid.")
	}
	if seenLoserImp1 {
		t.Errorf("foreachBestBid should not execute on apn backup bid.")
	}
	if !seenWinnerImp2 {
		t.Errorf("foreachBestBid did not execute on rubicon winning bid.")
	}
	if !seenLoserImp2 {
		t.Errorf("foreachBestBid did not execute on apn best-effort losing bid.")
	}

	if numBestBids != 3 {
		t.Errorf("expected 3 best-effort bids. Got %d", numBestBids)
	}
}
