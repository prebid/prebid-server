package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func Test_updateContentObjectForBidder(t *testing.T) {

	createBidderRequest := func(BidRequest *openrtb2.BidRequest) []BidderRequest {
		newReq := *BidRequest
		newReq.ID = "2"
		return []BidderRequest{{
			BidderName: "pubmatic",
			BidRequest: BidRequest,
		},
			{
				BidderName: "appnexus",
				BidRequest: &newReq,
			},
		}
	}

	type args struct {
		BidRequest *openrtb2.BidRequest
		requestExt *openrtb_ext.ExtRequest
	}
	tests := []struct {
		name                    string
		args                    args
		wantedAllBidderRequests []BidderRequest
	}{
		{
			name: "No Transparency Object",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
			},
		},
		{
			name: "No Content Object in App/Site",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
					},
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Site1",
					},
				},

				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Site1",
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Site1",
						},
					},
				},
			},
		},
		{
			name: "No partner/ default rules in tranpsarency",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Test",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Include All keys for bidder",
			args: args{

				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Test",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
									Keys:    []string{},
								},
								"appnexus": {
									Include: false,
									Keys:    []string{},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
						},
					},
				},
			},
		},
		{
			name: "Exclude All keys for pubmatic bidder",
			args: args{

				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: false,
									Keys:    []string{},
								},
								"appnexus": {
									Include: true,
									Keys:    []string{},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Include title field for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
									Keys:    []string{"title"},
								},
								"appnexus": {
									Include: false,
									Keys:    []string{"genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
			},
		},
		{
			name: "Exclude title field for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: false,
									Keys:    []string{"title"},
								},
								"appnexus": {
									Include: true,
									Keys:    []string{"genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Use default rule for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title:    "Title1",
							Genre:    "Genre1",
							Series:   "Series1",
							Season:   "Season1",
							Artist:   "Artist1",
							Album:    "Album1",
							ISRC:     "isrc1",
							Producer: &openrtb2.Producer{},
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"default": {
									Include: true,
									Keys: []string{
										"id", "episode", "series", "season", "artist", "genre", "album", "isrc", "producer", "url", "cat", "prodq", "videoquality", "context", "contentrating", "userrating", "qagmediarating", "livestream", "sourcerelationship", "len", "language", "embeddable", "data", "ext"},
								},
								"pubmatic": {
									Include: true,
									Keys:    []string{"title", "genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre:    "Genre1",
								Series:   "Series1",
								Season:   "Season1",
								Artist:   "Artist1",
								Album:    "Album1",
								ISRC:     "isrc1",
								Producer: &openrtb2.Producer{},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allBidderRequests := createBidderRequest(tt.args.BidRequest)
			updateContentObjectForBidder(allBidderRequests, tt.args.requestExt)
			assert.Equal(t, tt.wantedAllBidderRequests, allBidderRequests, tt.name)
		})
	}
}

func Benchmark_updateContentObjectForBidder(b *testing.B) {

	createBidderRequest := func(BidRequest *openrtb2.BidRequest) []BidderRequest {
		newReq := *BidRequest
		newReq.ID = "2"
		return []BidderRequest{{
			BidderName: "pubmatic",
			BidRequest: BidRequest,
		},
			{
				BidderName: "appnexus",
				BidRequest: &newReq,
			},
		}
	}

	type args struct {
		BidRequest *openrtb2.BidRequest
		requestExt *openrtb_ext.ExtRequest
	}
	tests := []struct {
		name                    string
		args                    args
		wantedAllBidderRequests []BidderRequest
	}{
		{
			name: "No Transparency Object",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
			},
		},
		{
			name: "No Content Object in App/Site",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
					},
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Site1",
					},
				},

				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Site1",
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Site1",
						},
					},
				},
			},
		},
		{
			name: "No partner/ default rules in tranpsarency",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Test",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Include All keys for bidder",
			args: args{

				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					Site: &openrtb2.Site{
						ID:   "1",
						Name: "Test",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
									Keys:    []string{},
								},
								"appnexus": {
									Include: false,
									Keys:    []string{},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						Site: &openrtb2.Site{
							ID:   "1",
							Name: "Test",
						},
					},
				},
			},
		},
		{
			name: "Exclude All keys for pubmatic bidder",
			args: args{

				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: false,
									Keys:    []string{},
								},
								"appnexus": {
									Include: true,
									Keys:    []string{},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Include title field for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: true,
									Keys:    []string{"title"},
								},
								"appnexus": {
									Include: false,
									Keys:    []string{"genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
							},
						},
					},
				},
			},
		},
		{
			name: "Exclude title field for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title: "Title1",
							Genre: "Genre1",
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"pubmatic": {
									Include: false,
									Keys:    []string{"title"},
								},
								"appnexus": {
									Include: true,
									Keys:    []string{"genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre: "Genre1",
							},
						},
					},
				},
			},
		},
		{
			name: "Use default rule for pubmatic bidder",
			args: args{
				BidRequest: &openrtb2.BidRequest{
					ID: "1",
					App: &openrtb2.App{
						ID:     "1",
						Name:   "Test",
						Bundle: "com.pubmatic.app",
						Content: &openrtb2.Content{
							Title:    "Title1",
							Genre:    "Genre1",
							Series:   "Series1",
							Season:   "Season1",
							Artist:   "Artist1",
							Album:    "Album1",
							ISRC:     "isrc1",
							Producer: &openrtb2.Producer{},
						},
					},
				},
				requestExt: &openrtb_ext.ExtRequest{
					Prebid: openrtb_ext.ExtRequestPrebid{
						Transparency: &openrtb_ext.TransparencyExt{
							Content: map[string]openrtb_ext.TransparencyRule{
								"default": {
									Include: true,
									Keys: []string{
										"id", "episode", "series", "season", "artist", "genre", "album", "isrc", "producer", "url", "cat", "prodq", "videoquality", "context", "contentrating", "userrating", "qagmediarating", "livestream", "sourcerelationship", "len", "language", "embeddable", "data", "ext"},
								},
								"pubmatic": {
									Include: true,
									Keys:    []string{"title", "genre"},
								},
							},
						},
					},
				},
			},
			wantedAllBidderRequests: []BidderRequest{
				{
					BidderName: "pubmatic",
					BidRequest: &openrtb2.BidRequest{
						ID: "1",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Title: "Title1",
								Genre: "Genre1",
							},
						},
					},
				},
				{
					BidderName: "appnexus",
					BidRequest: &openrtb2.BidRequest{
						ID: "2",
						App: &openrtb2.App{
							ID:     "1",
							Name:   "Test",
							Bundle: "com.pubmatic.app",
							Content: &openrtb2.Content{
								Genre:    "Genre1",
								Series:   "Series1",
								Season:   "Season1",
								Artist:   "Artist1",
								Album:    "Album1",
								ISRC:     "isrc1",
								Producer: &openrtb2.Producer{},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			allBidderRequests := createBidderRequest(tt.args.BidRequest)
			for i := 0; i < b.N; i++ {
				updateContentObjectForBidder(allBidderRequests, tt.args.requestExt)
			}
			//assert.Equal(t, tt.wantedAllBidderRequests, allBidderRequests, tt.name)
		})
	}
}
