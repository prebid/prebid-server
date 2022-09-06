package exchange

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/vastbidder"
	"github.com/prebid/prebid-server/config"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

//TestApplyAdvertiserBlocking verifies advertiser blocking
//Currently it is expected to work only with TagBidders and not woth
// normal bidders
func TestApplyAdvertiserBlocking(t *testing.T) {
	type args struct {
		advBlockReq     *openrtb2.BidRequest
		adaptorSeatBids map[*bidderAdapter]*pbsOrtbSeatBid // bidder adaptor and its dummy seat bids map
	}
	type want struct {
		rejectedBidIds       []string
		validBidCountPerSeat map[string]int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "reject_bid_of_blocked_adv_from_tag_bidder",
			args: args{
				advBlockReq: &openrtb2.BidRequest{
					BAdv: []string{"a.com"}, // block bids returned by a.com
				},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("vast_tag_bidder"): { // tag bidder returning 1 bid from blocked advertiser
						bids: []*pbsOrtbBid{
							{
								bid: &openrtb2.Bid{
									ID:      "a.com_bid",
									ADomain: []string{"a.com"},
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:      "b.com_bid",
									ADomain: []string{"b.com"},
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:      "keep_ba.com",
									ADomain: []string{"ba.com"},
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:      "keep_ba.com",
									ADomain: []string{"b.a.com.shri.com"},
								},
							},
							{
								bid: &openrtb2.Bid{
									ID:      "reject_b.a.com.a.com.b.c.d.a.com",
									ADomain: []string{"b.a.com.a.com.b.c.d.a.com"},
								},
							},
						},
						bidderCoreName: openrtb_ext.BidderVASTBidder,
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"a.com_bid", "reject_b.a.com.a.com.b.c.d.a.com"},
				validBidCountPerSeat: map[string]int{
					"vast_tag_bidder": 3,
				},
			},
		},
		{
			name: "Badv_is_not_present", // expect no advertiser blocking
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: nil},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tab_bidder_1"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_1", ADomain: []string{"a.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_1"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{}, // no bid rejection expected
				validBidCountPerSeat: map[string]int{
					"tab_bidder_1": 2,
				},
			},
		},
		{
			name: "adomain_is_not_present_but_Badv_is_set", // reject bids without adomain as badv is set
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"advertiser_1.com"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_bidder_1"): {
						bids: []*pbsOrtbBid{ // expect all bids are rejected
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_1_without_adomain"}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_1_with_empty_adomain", ADomain: []string{"", " "}}},
						},
					},
					newTestRtbAdapter("rtb_bidder_1"): {
						bids: []*pbsOrtbBid{ // all bids should be present. It belongs to RTB adapator
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_2_without_adomain"}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_2_with_empty_adomain", ADomain: []string{"", " "}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"bid_1_adapter_1_without_adomain", "bid_2_adapter_1_with_empty_adomain"},
				validBidCountPerSeat: map[string]int{
					"tag_bidder_1": 0, // expect 0 bids. i.e. all bids are rejected
					"rtb_bidder_1": 2, // no bid must be rejected
				},
			},
		},
		{
			name: "adomain_and_badv_is_not_present", // expect no advertiser blocking
			args: args{
				advBlockReq: &openrtb2.BidRequest{},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_adaptor_1"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ID: "bid_without_adomain"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{}, // no rejection expected as badv not present
				validBidCountPerSeat: map[string]int{
					"tag_adaptor_1": 1,
				},
			},
		},
		{
			name: "empty_badv", // expect no advertiser blocking
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_bidder_1"): {
						bids: []*pbsOrtbBid{ // expect all bids are rejected
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_1", ADomain: []string{"a.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_1"}},
						},
					},
					newTestRtbAdapter("rtb_bidder_1"): {
						bids: []*pbsOrtbBid{ // all bids should be present. It belongs to RTB adapator
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_2_without_adomain"}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_2_with_empty_adomain", ADomain: []string{"", " "}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{}, // no rejections expect as there is not badv set
				validBidCountPerSeat: map[string]int{
					"tag_bidder_1": 2,
					"rtb_bidder_1": 2,
				},
			},
		},
		{
			name: "nil_badv", // expect no advertiser blocking
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: nil},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_bidder_1"): {
						bids: []*pbsOrtbBid{ // expect all bids are rejected
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_1", ADomain: []string{"a.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_1"}},
						},
					},
					newTestRtbAdapter("rtb_bidder_1"): {
						bids: []*pbsOrtbBid{ // all bids should be present. It belongs to RTB adapator
							{bid: &openrtb2.Bid{ID: "bid_1_adapter_2_without_adomain"}},
							{bid: &openrtb2.Bid{ID: "bid_2_adapter_2_with_empty_adomain", ADomain: []string{"", " "}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{}, // no rejections expect as there is not badv set
				validBidCountPerSeat: map[string]int{
					"tag_bidder_1": 2,
					"rtb_bidder_1": 2,
				},
			},
		},
		{
			name: "ad_domains_normalized_and_checked",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"a.com"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("my_adapter"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ID: "bid_1_of_blocked_adv", ADomain: []string{"www.a.com"}}},
							// expect a.com is extracted from page url
							{bid: &openrtb2.Bid{ID: "bid_2_of_blocked_adv", ADomain: []string{"http://a.com/my/page?k1=v1&k2=v2"}}},
							// invalid adomain - will be skipped and the bid will be not be rejected
							{bid: &openrtb2.Bid{ID: "bid_3_with_domain_abcd1234", ADomain: []string{"abcd1234"}}},
						},
					}},
			},
			want: want{
				rejectedBidIds:       []string{"bid_1_of_blocked_adv", "bid_2_of_blocked_adv"},
				validBidCountPerSeat: map[string]int{"my_adapter": 1},
			},
		}, {
			name: "multiple_badv",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"advertiser_1.com", "advertiser_2.com", "www.advertiser_3.com"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_adapter_1"): {
						bids: []*pbsOrtbBid{
							// adomain without www prefix
							{bid: &openrtb2.Bid{ID: "bid_1_tag_adapter_1", ADomain: []string{"advertiser_3.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_tag_adapter_1", ADomain: []string{"advertiser_2.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_3_tag_adapter_1", ADomain: []string{"advertiser_4.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_4_tag_adapter_1", ADomain: []string{"advertiser_100.com"}}},
						},
					},
					newTestTagAdapter("tag_adapter_2"): {
						bids: []*pbsOrtbBid{
							// adomain has www prefix
							{bid: &openrtb2.Bid{ID: "bid_1_tag_adapter_2", ADomain: []string{"www.advertiser_1.com"}}},
						},
					},
					newTestRtbAdapter("rtb_adapter_1"): {
						bids: []*pbsOrtbBid{
							// should not reject following bid though its advertiser is blocked
							// because this bid belongs to RTB Adaptor
							{bid: &openrtb2.Bid{ID: "bid_1_rtb_adapter_2", ADomain: []string{"advertiser_1.com"}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"bid_1_tag_adapter_1", "bid_2_tag_adapter_1", "bid_1_tag_adapter_2"},
				validBidCountPerSeat: map[string]int{
					"tag_adapter_1": 2,
					"tag_adapter_2": 0,
					"rtb_adapter_1": 1,
				},
			},
		}, {
			name: "multiple_adomain",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"www.advertiser_3.com"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_adapter_1"): {
						bids: []*pbsOrtbBid{
							// adomain without www prefix
							{bid: &openrtb2.Bid{ID: "bid_1_tag_adapter_1", ADomain: []string{"a.com", "b.com", "advertiser_3.com", "d.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_tag_adapter_1", ADomain: []string{"a.com", "https://advertiser_3.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_3_tag_adapter_1", ADomain: []string{"advertiser_4.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_4_tag_adapter_1", ADomain: []string{"advertiser_100.com"}}},
						},
					},
					newTestTagAdapter("tag_adapter_2"): {
						bids: []*pbsOrtbBid{
							// adomain has www prefix
							{bid: &openrtb2.Bid{ID: "bid_1_tag_adapter_2", ADomain: []string{"a.com", "b.com", "www.advertiser_3.com"}}},
						},
					},
					newTestRtbAdapter("rtb_adapter_1"): {
						bids: []*pbsOrtbBid{
							// should not reject following bid though its advertiser is blocked
							// because this bid belongs to RTB Adaptor
							{bid: &openrtb2.Bid{ID: "bid_1_rtb_adapter_2", ADomain: []string{"a.com", "b.com", "advertiser_3.com"}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"bid_1_tag_adapter_1", "bid_2_tag_adapter_1", "bid_1_tag_adapter_2"},
				validBidCountPerSeat: map[string]int{
					"tag_adapter_1": 2,
					"tag_adapter_2": 0,
					"rtb_adapter_1": 1,
				},
			},
		}, {
			name: "case_insensitive_badv", // case of domain not matters
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"ADVERTISER_1.COM"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_adapter_1"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ID: "bid_1_rtb_adapter_1", ADomain: []string{"advertiser_1.com"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_rtb_adapter_1", ADomain: []string{"www.advertiser_1.com"}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"bid_1_rtb_adapter_1", "bid_2_rtb_adapter_1"},
				validBidCountPerSeat: map[string]int{
					"tag_adapter_1": 0, // expect all bids are rejected as belongs to blocked advertiser
				},
			},
		},
		{
			name: "case_insensitive_adomain",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"advertiser_1.com"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_adapter_1"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ID: "bid_1_rtb_adapter_1", ADomain: []string{"advertiser_1.COM"}}},
							{bid: &openrtb2.Bid{ID: "bid_2_rtb_adapter_1", ADomain: []string{"wWw.ADVERTISER_1.com"}}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"bid_1_rtb_adapter_1", "bid_2_rtb_adapter_1"},
				validBidCountPerSeat: map[string]int{
					"tag_adapter_1": 0, // expect all bids are rejected as belongs to blocked advertiser
				},
			},
		},
		{
			name: "various_tld_combinations",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"http://blockme.shri"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("block_bidder"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ADomain: []string{"www.blockme.shri"}, ID: "reject_www.blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"http://www.blockme.shri"}, ID: "rejecthttp://www.blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"https://blockme.shri"}, ID: "reject_https://blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"https://www.blockme.shri"}, ID: "reject_https://www.blockme.shri"}},
						},
					},
					newTestRtbAdapter("rtb_non_block_bidder"): {
						bids: []*pbsOrtbBid{ // all below bids are eligible and should not be rejected
							{bid: &openrtb2.Bid{ADomain: []string{"www.blockme.shri"}, ID: "accept_bid_www.blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"http://www.blockme.shri"}, ID: "accept_bid__http://www.blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"https://blockme.shri"}, ID: "accept_bid__https://blockme.shri"}},
							{bid: &openrtb2.Bid{ADomain: []string{"https://www.blockme.shri"}, ID: "accept_bid__https://www.blockme.shri"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"reject_www.blockme.shri", "reject_http://www.blockme.shri", "reject_https://blockme.shri", "reject_https://www.blockme.shri"},
				validBidCountPerSeat: map[string]int{
					"block_bidder":         0,
					"rtb_non_block_bidder": 4,
				},
			},
		},
		{
			name: "subdomain_tests",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"10th.college.puneunv.edu"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("block_bidder"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ADomain: []string{"shri.10th.college.puneunv.edu"}, ID: "reject_shri.10th.college.puneunv.edu"}},
							{bid: &openrtb2.Bid{ADomain: []string{"puneunv.edu"}, ID: "allow_puneunv.edu"}},
							{bid: &openrtb2.Bid{ADomain: []string{"http://WWW.123.456.10th.college.PUNEUNV.edu"}, ID: "reject_123.456.10th.college.puneunv.edu"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{"reject_shri.10th.college.puneunv.edu", "reject_123.456.10th.college.puneunv.edu"},
				validBidCountPerSeat: map[string]int{
					"block_bidder": 1,
				},
			},
		}, {
			name: "only_domain_test", // do not expect bid rejection. edu is valid domain
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"edu"}},
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_bidder"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ADomain: []string{"school.edu"}, ID: "keep_bid_school.edu"}},
							{bid: &openrtb2.Bid{ADomain: []string{"edu"}, ID: "keep_bid_edu"}},
							{bid: &openrtb2.Bid{ADomain: []string{"..edu"}, ID: "keep_bid_..edu"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{},
				validBidCountPerSeat: map[string]int{
					"tag_bidder": 3,
				},
			},
		},
		{
			name: "public_suffix_in_badv",
			args: args{
				advBlockReq: &openrtb2.BidRequest{BAdv: []string{"co.in"}}, // co.in is valid public suffix
				adaptorSeatBids: map[*bidderAdapter]*pbsOrtbSeatBid{
					newTestTagAdapter("tag_bidder"): {
						bids: []*pbsOrtbBid{
							{bid: &openrtb2.Bid{ADomain: []string{"a.co.in"}, ID: "allow_a.co.in"}},
							{bid: &openrtb2.Bid{ADomain: []string{"b.com"}, ID: "allow_b.com"}},
						},
					},
				},
			},
			want: want{
				rejectedBidIds: []string{},
				validBidCountPerSeat: map[string]int{
					"tag_bidder": 2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "reject_bid_of_blocked_adv_from_tag_bidder" {
				return
			}
			seatBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)
			tagBidders := make(map[openrtb_ext.BidderName]adapters.Bidder)
			adapterMap := make(map[openrtb_ext.BidderName]AdaptedBidder, 0)
			for adaptor, sbids := range tt.args.adaptorSeatBids {
				adapterMap[adaptor.BidderName] = adaptor
				if tagBidder, ok := adaptor.Bidder.(*vastbidder.TagBidder); ok {
					tagBidders[adaptor.BidderName] = tagBidder
				}
				seatBids[adaptor.BidderName] = sbids
			}

			// applyAdvertiserBlocking internally uses tagBidders from (adapter_map.go)
			// not testing alias here
			seatBids, rejections := applyAdvertiserBlocking(tt.args.advBlockReq, seatBids)

			re := regexp.MustCompile("bid rejected \\[bid ID:(.*?)\\] reason")
			for bidder, sBid := range seatBids {
				// verify only eligible bids are returned
				assert.Equal(t, tt.want.validBidCountPerSeat[string(bidder)], len(sBid.bids), "Expected eligible bids are %d, but found [%d] ", tt.want.validBidCountPerSeat[string(bidder)], len(sBid.bids))
				// verify  rejections
				assert.Equal(t, len(tt.want.rejectedBidIds), len(rejections), "Expected bid rejections are %d, but found [%d]", len(tt.want.rejectedBidIds), len(rejections))
				// verify rejected bid ids
				present := false
				for _, expectRejectedBidID := range tt.want.rejectedBidIds {
					for _, rejection := range rejections {
						match := re.FindStringSubmatch(rejection)
						rejectedBidID := strings.Trim(match[1], " ")
						if expectRejectedBidID == rejectedBidID {
							present = true
							break
						}
					}
					if present {
						break
					}
				}
				if len(tt.want.rejectedBidIds) > 0 && !present {
					assert.Fail(t, "Expected Bid ID [%s] as rejected. But bid is not rejected", re)
				}

				if sBid.bidderCoreName != openrtb_ext.BidderVASTBidder {
					continue // advertiser blocking is currently enabled only for tag bidders
				}
				// verify eligible bids not belongs to blocked advertisers
				for _, bid := range sBid.bids {
					if nil != bid.bid.ADomain {
						for _, adomain := range bid.bid.ADomain {
							for _, blockDomain := range tt.args.advBlockReq.BAdv {
								nDomain, _ := normalizeDomain(adomain)
								if nDomain == blockDomain {
									assert.Fail(t, "bid %s with ad domain %s is not blocked", bid.bid.ID, adomain)
								}
							}
						}
					}

					// verify this bid not belongs to rejected list
					for _, rejectedBidID := range tt.want.rejectedBidIds {
						if rejectedBidID == bid.bid.ID {
							assert.Fail(t, "Bid ID [%s] is not expected in list of rejected bids", bid.bid.ID)
						}
					}
				}
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	type args struct {
		domain string
	}
	type want struct {
		domain string
		err    error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "a.com", args: args{domain: "a.com"}, want: want{domain: "a.com"}},
		{name: "http://a.com", args: args{domain: "http://a.com"}, want: want{domain: "a.com"}},
		{name: "https://a.com", args: args{domain: "https://a.com"}, want: want{domain: "a.com"}},
		{name: "https://www.a.com", args: args{domain: "https://www.a.com"}, want: want{domain: "a.com"}},
		{name: "https://www.a.com/my/page?k=1", args: args{domain: "https://www.a.com/my/page?k=1"}, want: want{domain: "a.com"}},
		{name: "empty_domain", args: args{domain: ""}, want: want{domain: ""}},
		{name: "trim_domain", args: args{domain: " trim.me?k=v    "}, want: want{domain: "trim.me"}},
		{name: "trim_domain_with_http_in_it", args: args{domain: " http://trim.me?k=v    "}, want: want{domain: "trim.me"}},
		{name: "https://www.something.a.com/my/page?k=1", args: args{domain: "https://www.something.a.com/my/page?k=1"}, want: want{domain: "something.a.com"}},
		{name: "wWW.something.a.com", args: args{domain: "wWW.something.a.com"}, want: want{domain: "something.a.com"}},
		{name: "2_times_www", args: args{domain: "www.something.www.a.com"}, want: want{domain: "something.www.a.com"}},
		{name: "consecutive_www", args: args{domain: "www.www.something.a.com"}, want: want{domain: "www.something.a.com"}},
		{name: "abchttp.com", args: args{domain: "abchttp.com"}, want: want{domain: "abchttp.com"}},
		{name: "HTTP://CAPS.com", args: args{domain: "HTTP://CAPS.com"}, want: want{domain: "caps.com"}},

		// publicsuffix
		{name: "co.in", args: args{domain: "co.in"}, want: want{domain: "", err: fmt.Errorf("domain [co.in] is public suffix")}},
		{name: ".co.in", args: args{domain: ".co.in"}, want: want{domain: ".co.in"}},
		{name: "amazon.co.in", args: args{domain: "amazon.co.in"}, want: want{domain: "amazon.co.in"}},
		// we wont check if shriprasad belongs to icann
		{name: "shriprasad", args: args{domain: "shriprasad"}, want: want{domain: "", err: fmt.Errorf("domain [shriprasad] is public suffix")}},
		{name: ".shriprasad", args: args{domain: ".shriprasad"}, want: want{domain: ".shriprasad"}},
		{name: "abc.shriprasad", args: args{domain: "abc.shriprasad"}, want: want{domain: "abc.shriprasad"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjustedDomain, err := normalizeDomain(tt.args.domain)
			actualErr := "nil"
			expectedErr := "nil"
			if nil != err {
				actualErr = err.Error()
			}
			if nil != tt.want.err {
				actualErr = tt.want.err.Error()
			}
			assert.Equal(t, tt.want.err, err, "Expected error is %s, but found [%s]", expectedErr, actualErr)
			assert.Equal(t, tt.want.domain, adjustedDomain, "Expected domain is %s, but found [%s]", tt.want.domain, adjustedDomain)
		})
	}
}

func newTestTagAdapter(name string) *bidderAdapter {
	return &bidderAdapter{
		Bidder:     vastbidder.NewTagBidder(openrtb_ext.BidderName(name), config.Adapter{}),
		BidderName: openrtb_ext.BidderName(name),
	}
}

func newTestRtbAdapter(name string) *bidderAdapter {
	return &bidderAdapter{
		Bidder:     &goodSingleBidder{},
		BidderName: openrtb_ext.BidderName(name),
	}
}

func TestRecordAdaptorDuplicateBidIDs(t *testing.T) {
	type bidderCollisions = map[string]int
	testCases := []struct {
		scenario         string
		bidderCollisions *bidderCollisions // represents no of collisions detected for bid.id at bidder level for given request
		hasCollision     bool
	}{
		{scenario: "invalid collision value", bidderCollisions: &map[string]int{"bidder-1": -1}, hasCollision: false},
		{scenario: "no collision", bidderCollisions: &map[string]int{"bidder-1": 0}, hasCollision: false},
		{scenario: "one collision", bidderCollisions: &map[string]int{"bidder-1": 1}, hasCollision: false},
		{scenario: "multiple collisions", bidderCollisions: &map[string]int{"bidder-1": 2}, hasCollision: true}, // when 2 collisions it counter will be 1
		{scenario: "multiple bidders", bidderCollisions: &map[string]int{"bidder-1": 2, "bidder-2": 4}, hasCollision: true},
		{scenario: "multiple bidders with bidder-1 no collision", bidderCollisions: &map[string]int{"bidder-1": 1, "bidder-2": 4}, hasCollision: true},
		{scenario: "no bidders", bidderCollisions: nil, hasCollision: false},
	}
	testEngine := metricsConf.NewMetricsEngine(&config.Configuration{}, nil, nil)

	for _, testcase := range testCases {
		var adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		if nil == testcase.bidderCollisions {
			break
		}
		adapterBids = make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)
		for bidder, collisions := range *testcase.bidderCollisions {
			bids := make([]*pbsOrtbBid, 0)
			testBidID := "bid_id_for_bidder_" + bidder
			// add bids as per collisions value
			bidCount := 0
			for ; bidCount < collisions; bidCount++ {
				bids = append(bids, &pbsOrtbBid{
					bid: &openrtb2.Bid{
						ID: testBidID,
					},
				})
			}
			if nil == adapterBids[openrtb_ext.BidderName(bidder)] {
				adapterBids[openrtb_ext.BidderName(bidder)] = new(pbsOrtbSeatBid)
			}
			adapterBids[openrtb_ext.BidderName(bidder)].bids = bids
		}
		assert.Equal(t, testcase.hasCollision, recordAdaptorDuplicateBidIDs(testEngine, adapterBids))
	}
}
