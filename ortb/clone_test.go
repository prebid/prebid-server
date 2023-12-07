package ortb

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestCloneApp(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneApp(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.App{}
		result := CloneApp(given)
		assert.Equal(t, given, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.App{
			ID:                     "anyID",
			Name:                   "anyName",
			Bundle:                 "anyBundle",
			Domain:                 "anyDomain",
			StoreURL:               "anyStoreURL",
			CatTax:                 adcom1.CatTaxIABContent10,
			Cat:                    []string{"cat1"},
			SectionCat:             []string{"sectionCat1"},
			PageCat:                []string{"pageCat1"},
			Ver:                    "anyVer",
			PrivacyPolicy:          1,
			Paid:                   2,
			Publisher:              &openrtb2.Publisher{ID: "anyPublisher", Ext: json.RawMessage(`{"publisher":1}`)},
			Content:                &openrtb2.Content{ID: "anyContent", Ext: json.RawMessage(`{"content":1}`)},
			Keywords:               "anyKeywords",
			KwArray:                []string{"key1"},
			InventoryPartnerDomain: "anyInventoryPartnerDomain",
			Ext:                    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneApp(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Cat, result.Cat, "cat")
		assert.NotSame(t, given.SectionCat, result.SectionCat, "sectioncat")
		assert.NotSame(t, given.PageCat, result.PageCat, "pagecat")
		assert.NotSame(t, given.Publisher, result.Publisher, "publisher")
		assert.NotSame(t, given.Publisher.Ext, result.Publisher.Ext, "publisher-ext")
		assert.NotSame(t, given.Content, result.Content, "content")
		assert.NotSame(t, given.Content.Ext, result.Content.Ext, "content-ext")
		assert.NotSame(t, given.KwArray, result.KwArray, "kwarray")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.App{})),
			[]string{
				"Cat",
				"SectionCat",
				"PageCat",
				"Publisher",
				"Content",
				"KwArray",
				"Ext",
			})
	})
}

func TestClonePublisher(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := ClonePublisher(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Publisher{}
		result := ClonePublisher(given)
		assert.Equal(t, given, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Publisher{
			ID:     "anyID",
			Name:   "anyName",
			CatTax: adcom1.CatTaxIABContent20,
			Cat:    []string{"cat1"},
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := ClonePublisher(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Cat, result.Cat, "cat")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Publisher{})),
			[]string{
				"Cat",
				"Ext",
			})
	})
}

func TestCloneContent(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneContent(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Content{}
		result := CloneContent(given)
		assert.Equal(t, given, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Content{
			ID:                 "anyID",
			Episode:            1,
			Title:              "anyTitle",
			Series:             "anySeries",
			Season:             "anySeason",
			Artist:             "anyArtist",
			Genre:              "anyGenre",
			Album:              "anyAlbum",
			ISRC:               "anyIsrc",
			Producer:           &openrtb2.Producer{ID: "anyID", Cat: []string{"anyCat"}},
			URL:                "anyUrl",
			CatTax:             adcom1.CatTaxIABContent10,
			Cat:                []string{"cat1"},
			ProdQ:              ptrutil.ToPtr(adcom1.ProductionProsumer),
			VideoQuality:       ptrutil.ToPtr(adcom1.ProductionProfessional),
			Context:            adcom1.ContentApp,
			ContentRating:      "anyContentRating",
			UserRating:         "anyUserRating",
			QAGMediaRating:     adcom1.MediaRatingAll,
			Keywords:           "anyKeywords",
			KwArray:            []string{"key1"},
			LiveStream:         2,
			SourceRelationship: 3,
			Len:                4,
			Language:           "anyLanguage",
			LangB:              "anyLangB",
			Embeddable:         5,
			Data:               []openrtb2.Data{{ID: "1", Ext: json.RawMessage(`{"data":1}`)}},
			Network:            &openrtb2.Network{ID: "anyNetwork", Ext: json.RawMessage(`{"network":1}`)},
			Channel:            &openrtb2.Channel{ID: "anyChannel", Ext: json.RawMessage(`{"channel":1}`)},
			Ext:                json.RawMessage(`{"anyField":1}`),
		}
		result := CloneContent(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Producer, result.Producer, "producer")
		assert.NotSame(t, given.Producer.Cat, result.Producer.Cat, "producer-cat")
		assert.NotSame(t, given.Cat, result.Cat, "cat")
		assert.NotSame(t, given.ProdQ, result.ProdQ, "prodq")
		assert.NotSame(t, given.VideoQuality, result.VideoQuality, "videoquality")
		assert.NotSame(t, given.KwArray, result.KwArray, "kwarray")
		assert.NotSame(t, given.Data, result.Data, "data")
		assert.NotSame(t, given.Data[0], result.Data[0], "data-item")
		assert.NotSame(t, given.Data[0].Ext, result.Data[0].Ext, "data-item-ext")
		assert.NotSame(t, given.Network, result.Network, "network")
		assert.NotSame(t, given.Network.Ext, result.Network.Ext, "network-ext")
		assert.NotSame(t, given.Channel, result.Channel, "channel")
		assert.NotSame(t, given.Channel.Ext, result.Channel.Ext, "channel-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Content{})),
			[]string{
				"Producer",
				"Cat",
				"ProdQ",
				"VideoQuality",
				"KwArray",
				"Data",
				"Network",
				"Channel",
				"Ext",
			})
	})
}

func TestCloneProducer(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneProducer(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Producer{}
		result := CloneProducer(given)
		assert.Equal(t, given, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Producer{
			ID:     "anyID",
			Name:   "anyName",
			CatTax: adcom1.CatTaxIABContent20,
			Cat:    []string{"cat1"},
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneProducer(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Cat, result.Cat, "cat")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Producer{})),
			[]string{
				"Cat",
				"Ext",
			})
	})
}

func TestCloneDataSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneDataSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.Data{}
		result := CloneDataSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.Data{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneDataSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.Data{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
			{ID: "2", Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneDataSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
	})
}

func TestCloneData(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.Data{
			ID:      "anyID",
			Name:    "anyName",
			Segment: []openrtb2.Segment{{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)}},
			Ext:     json.RawMessage(`{"anyField":1}`),
		}
		result := CloneData(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Segment, result.Segment, "segment")
		assert.NotSame(t, given.Segment[0], result.Segment[0], "segment-item")
		assert.NotSame(t, given.Segment[0].Ext, result.Segment[0].Ext, "segment-item-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Data{})),
			[]string{
				"Segment",
				"Ext",
			})
	})
}

func TestCloneSegmentSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneSegmentSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.Segment{}
		result := CloneSegmentSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.Segment{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneSegmentSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.Segment{
			{Ext: json.RawMessage(`{"anyField":1}`)},
			{Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneSegmentSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
	})
}

func TestCloneSegment(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.Segment{
			ID:    "anyID",
			Name:  "anyName",
			Value: "anyValue",
			Ext:   json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSegment(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Segment{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneNetwork(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneNetwork(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Network{}
		result := CloneNetwork(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Network{
			ID:     "anyID",
			Name:   "anyName",
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneNetwork(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Network{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneChannel(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneChannel(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Channel{}
		result := CloneChannel(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Channel{
			ID:     "anyID",
			Name:   "anyName",
			Domain: "anyDomain",
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneChannel(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Channel{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneSite(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneSite(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Site{}
		result := CloneSite(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Site{
			ID:                     "anyID",
			Name:                   "anyName",
			Domain:                 "anyDomain",
			CatTax:                 adcom1.CatTaxIABContent10,
			Cat:                    []string{"cat1"},
			SectionCat:             []string{"sectionCat1"},
			PageCat:                []string{"pageCat1"},
			Page:                   "anyPage",
			Ref:                    "anyRef",
			Search:                 "anySearch",
			Mobile:                 1,
			PrivacyPolicy:          2,
			Publisher:              &openrtb2.Publisher{ID: "anyPublisher", Ext: json.RawMessage(`{"publisher":1}`)},
			Content:                &openrtb2.Content{ID: "anyContent", Ext: json.RawMessage(`{"content":1}`)},
			Keywords:               "anyKeywords",
			KwArray:                []string{"key1"},
			InventoryPartnerDomain: "anyInventoryPartnerDomain",
			Ext:                    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSite(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Cat, result.Cat, "cat")
		assert.NotSame(t, given.SectionCat, result.SectionCat, "sectioncat")
		assert.NotSame(t, given.PageCat, result.PageCat, "pagecat")
		assert.NotSame(t, given.Publisher, result.Publisher, "publisher")
		assert.NotSame(t, given.Publisher.Ext, result.Publisher.Ext, "publisher-ext")
		assert.NotSame(t, given.Content, result.Content, "content")
		assert.NotSame(t, given.Content.Ext, result.Content.Ext, "content-ext")
		assert.NotSame(t, given.KwArray, result.KwArray, "kwarray")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Site{})),
			[]string{
				"Cat",
				"SectionCat",
				"PageCat",
				"Publisher",
				"Content",
				"KwArray",
				"Ext",
			})
	})
}

func TestCloneUser(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneUser(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.User{}
		result := CloneUser(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.User{
			ID:         "anyID",
			BuyerUID:   "anyBuyerUID",
			Yob:        1,
			Gender:     "anyGender",
			Keywords:   "anyKeywords",
			KwArray:    []string{"key1"},
			CustomData: "anyCustomData",
			Geo:        &openrtb2.Geo{Lat: 1.2, Lon: 2.3, Ext: json.RawMessage(`{"geo":1}`)},
			Data:       []openrtb2.Data{{ID: "1", Ext: json.RawMessage(`{"data":1}`)}},
			Consent:    "anyConsent",
			EIDs:       []openrtb2.EID{{Source: "1", Ext: json.RawMessage(`{"eid":1}`)}},
			Ext:        json.RawMessage(`{"anyField":1}`),
		}
		result := CloneUser(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.KwArray, result.KwArray, "cat")
		assert.NotSame(t, given.Geo, result.Geo, "geo")
		assert.NotSame(t, given.Geo.Ext, result.Geo.Ext, "geo-ext")
		assert.NotSame(t, given.Data[0], result.Data[0], "data-item")
		assert.NotSame(t, given.Data[0].Ext, result.Data[0].Ext, "data-item-ext")
		assert.NotSame(t, given.EIDs[0], result.EIDs[0], "eids-item")
		assert.NotSame(t, given.EIDs[0].Ext, result.EIDs[0].Ext, "eids-item-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.User{})),
			[]string{
				"KwArray",
				"Geo",
				"Data",
				"EIDs",
				"Ext",
			})
	})
}

func TestCloneDevice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneDevice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Device{}
		result := CloneDevice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		var n int8 = 1
		np := &n
		ct := adcom1.ConnectionWIFI

		given := &openrtb2.Device{
			Geo:            &openrtb2.Geo{Lat: 1.2, Lon: 2.3, Ext: json.RawMessage(`{"geo":1}`)},
			DNT:            np,
			Lmt:            np,
			UA:             "UserAgent",
			SUA:            &openrtb2.UserAgent{Mobile: np, Model: "iPad"},
			IP:             "127.0.0.1",
			IPv6:           "2001::",
			DeviceType:     adcom1.DeviceTablet,
			Make:           "Apple",
			Model:          "iPad",
			OS:             "macOS",
			OSV:            "1.2.3",
			HWV:            "mini",
			H:              20,
			W:              30,
			PPI:            100,
			PxRatio:        200,
			JS:             2,
			GeoFetch:       4,
			FlashVer:       "1.22.33",
			Language:       "En",
			LangB:          "ENG",
			Carrier:        "AT&T",
			MCCMNC:         "111-222",
			ConnectionType: &ct,
			IFA:            "IFA",
			DIDSHA1:        "DIDSHA1",
			DIDMD5:         "DIDMD5",
			DPIDSHA1:       "DPIDSHA1",
			DPIDMD5:        "DPIDMD5",
			MACSHA1:        "MACSHA1",
			MACMD5:         "MACMD5",
			Ext:            json.RawMessage(`{"anyField":1}`),
		}
		result := CloneDevice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Geo, result.Geo, "geo")
		assert.NotSame(t, given.Geo.Ext, result.Geo.Ext, "geo-ext")
		assert.NotSame(t, given.DNT, result.DNT, "dnt")
		assert.NotSame(t, given.Lmt, result.Lmt, "lmt")
		assert.NotSame(t, given.SUA, result.SUA, "sua")
		assert.NotSame(t, given.ConnectionType, result.ConnectionType, "connectionType")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Device{})),
			[]string{
				"Geo",
				"DNT",
				"Lmt",
				"SUA",
				"ConnectionType",
				"Ext",
			})
	})
}

func TestCloneInt8Pointer(t *testing.T) {

	t.Run("nil", func(t *testing.T) {
		result := CloneInt8Pointer(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		var given *int8
		result := CloneInt8Pointer(given)
		assert.Nil(t, result)
	})

	t.Run("populated", func(t *testing.T) {
		var n int8 = 1
		given := &n
		result := CloneInt8Pointer(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
	})
}

func TestCloneUserAgent(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneUserAgent(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.UserAgent{}
		result := CloneUserAgent(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		var n int8 = 1
		np := &n

		given := &openrtb2.UserAgent{
			Browsers:     []openrtb2.BrandVersion{{Brand: "Apple"}},
			Platform:     &openrtb2.BrandVersion{Brand: "Apple"},
			Mobile:       np,
			Architecture: "X86",
			Bitness:      "64",
			Model:        "iPad",
			Source:       adcom1.UASourceLowEntropy,
			Ext:          json.RawMessage(`{"anyField":1}`),
		}
		result := CloneUserAgent(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Browsers, result.Browsers, "browsers")
		assert.NotSame(t, given.Platform, result.Platform, "platform")
		assert.NotSame(t, given.Mobile, result.Mobile, "mobile")
		assert.NotSame(t, given.Architecture, result.Architecture, "architecture")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.UserAgent{})),
			[]string{
				"Browsers",
				"Platform",
				"Mobile",
				"Ext",
			})
	})
}

func TestCloneBrandVersionSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneBrandVersionSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.BrandVersion{}
		result := CloneBrandVersionSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.BrandVersion{
			{Brand: "1", Version: []string{"s1", "s2"}, Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneBrandVersionSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.BrandVersion{
			{Brand: "1", Version: []string{"s1", "s2"}, Ext: json.RawMessage(`{"anyField":1}`)},
			{Brand: "2", Version: []string{"s3", "s4"}, Ext: json.RawMessage(`{"anyField":1}`)},
			{Brand: "3", Version: []string{"s5", "s6"}, Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneBrandVersionSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
		assert.NotSame(t, given[2], result[2], "item1-pointer")
		assert.NotSame(t, given[2].Ext, result[2].Ext, "item1-pointer-ext")
	})
}

func TestCloneBrandVersion(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneBrandVersion(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.BrandVersion{}
		result := CloneBrandVersion(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.BrandVersion{
			Brand:   "Apple",
			Version: []string{"s1"},
			Ext:     json.RawMessage(`{"anyField":1}`),
		}
		result := CloneBrandVersion(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.BrandVersion{})),
			[]string{
				"Version",
				"Ext",
			})
	})
}

func TestCloneSource(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneSource(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Source{}
		result := CloneSource(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {

		given := &openrtb2.Source{
			FD:     1,
			TID:    "Tid",
			PChain: "PChain",
			SChain: &openrtb2.SupplyChain{
				Complete: 1,
				Nodes: []openrtb2.SupplyChainNode{
					{ASI: "asi", Ext: json.RawMessage(`{"anyField":1}`)},
				},
				Ext: json.RawMessage(`{"anyField":2}`),
			},
			Ext: json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSource(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.SChain, result.SChain, "schain")
		assert.NotSame(t, given.SChain.Ext, result.SChain.Ext, "schain.ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
		assert.NotSame(t, given.SChain.Nodes[0].Ext, result.SChain.Nodes[0].Ext, "schain.nodes.ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Source{})),
			[]string{
				"SChain",
				"Ext",
			})
	})
}

func TestCloneSChain(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneSource(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.SupplyChain{}
		result := CloneSChain(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.SupplyChain{
			Complete: 1,
			Nodes: []openrtb2.SupplyChainNode{
				{ASI: "asi", Ext: json.RawMessage(`{"anyField":1}`)},
			},
			Ext: json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSChain(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Nodes, result.Nodes, "nodes")
		assert.NotSame(t, given.Nodes[0].Ext, result.Nodes[0].Ext, "nodes.ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.SupplyChain{})),
			[]string{
				"Nodes",
				"Ext",
			})
	})
}

func TestCloneSupplyChainNodes(t *testing.T) {
	var n int8 = 1
	np := &n
	t.Run("nil", func(t *testing.T) {
		result := CloneSupplyChainNodes(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.SupplyChainNode{}
		result := CloneSupplyChainNodes(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.SupplyChainNode{
			{ASI: "asi", HP: np, Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneSupplyChainNodes(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].HP, result[0].HP, "item-pointer-hp")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.SupplyChainNode{
			{ASI: "asi", HP: np, Ext: json.RawMessage(`{"anyField":1}`)},
			{ASI: "asi", HP: np, Ext: json.RawMessage(`{"anyField":1}`)},
			{ASI: "asi", HP: np, Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneSupplyChainNodes(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[0].HP, result[0].HP, "item0-pointer-hp")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
		assert.NotSame(t, given[1].HP, result[1].HP, "item1-pointer-hp")
		assert.NotSame(t, given[2], result[2], "item2-pointer")
		assert.NotSame(t, given[2].Ext, result[2].Ext, "item2-pointer-ext")
		assert.NotSame(t, given[2].HP, result[2].HP, "item2-pointer-hp")
	})
}

func TestCloneSupplyChainNode(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		var n int8 = 1
		np := &n

		given := openrtb2.SupplyChainNode{
			ASI: "asi",
			HP:  np,
			Ext: json.RawMessage(`{"anyField":1}`),
		}
		result := CloneSupplyChainNode(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
		assert.NotSame(t, given.HP, result.HP, "hp")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.SupplyChainNode{})),
			[]string{
				"HP",
				"Ext",
			})
	})
}

func TestCloneGeo(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneGeo(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.Geo{}
		result := CloneGeo(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.Geo{
			Lat:           1.234,
			Lon:           5.678,
			Type:          adcom1.LocationGPS,
			Accuracy:      1,
			LastFix:       2,
			IPService:     adcom1.LocationServiceIP2Location,
			Country:       "anyCountry",
			Region:        "anyRegion",
			RegionFIPS104: "anyRegionFIPS104",
			Metro:         "anyMetro",
			City:          "anyCity",
			ZIP:           "anyZIP",
			UTCOffset:     3,
			Ext:           json.RawMessage(`{"anyField":1}`),
		}
		result := CloneGeo(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.Geo{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneEIDSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneEIDSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.EID{}
		result := CloneEIDSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.EID{
			{Source: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneEIDSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.EID{
			{Source: "1", Ext: json.RawMessage(`{"anyField":1}`)},
			{Source: "2", Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneEIDSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
	})
}

func TestCloneEID(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.EID{
			Source: "anySource",
			UIDs:   []openrtb2.UID{{ID: "1", Ext: json.RawMessage(`{"uid":1}`)}},
			Ext:    json.RawMessage(`{"anyField":1}`),
		}
		result := CloneEID(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.UIDs, result.UIDs, "uids")
		assert.NotSame(t, given.UIDs[0], result.UIDs[0], "uids-item")
		assert.NotSame(t, given.UIDs[0].Ext, result.UIDs[0].Ext, "uids-item-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.EID{})),
			[]string{
				"UIDs",
				"Ext",
			})
	})
}

func TestCloneUIDSlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneUIDSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := []openrtb2.UID{}
		result := CloneUIDSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.UID{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneUIDSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item-pointer-ext")
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.UID{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
			{ID: "2", Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneUIDSlice(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given[0], result[0], "item0-pointer")
		assert.NotSame(t, given[0].Ext, result[0].Ext, "item0-pointer-ext")
		assert.NotSame(t, given[1], result[1], "item1-pointer")
		assert.NotSame(t, given[1].Ext, result[1].Ext, "item1-pointer-ext")
	})
}

func TestCloneUID(t *testing.T) {
	t.Run("populated", func(t *testing.T) {
		given := openrtb2.UID{
			ID:    "anyID",
			AType: adcom1.AgentTypePerson,
			Ext:   json.RawMessage(`{"anyField":1}`),
		}
		result := CloneUID(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.UID{})),
			[]string{
				"Ext",
			})
	})
}

func TestCloneDOOH(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneDOOH(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.DOOH{}
		result := CloneDOOH(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.DOOH{
			ID:           "anyID",
			Name:         "anyName",
			VenueType:    []string{"venue1"},
			VenueTypeTax: ptrutil.ToPtr(adcom1.VenueTaxonomyAdCom),
			Publisher:    &openrtb2.Publisher{ID: "anyPublisher", Ext: json.RawMessage(`{"publisher":1}`)},
			Domain:       "anyDomain",
			Keywords:     "anyKeywords",
			Content:      &openrtb2.Content{ID: "anyContent", Ext: json.RawMessage(`{"content":1}`)},
			Ext:          json.RawMessage(`{"anyField":1}`),
		}
		result := CloneDOOH(given)
		assert.Equal(t, given, result, "equality")
		assert.NotSame(t, given, result, "pointer")
		assert.NotSame(t, given.VenueType, result.VenueType, "venuetype")
		assert.NotSame(t, given.VenueTypeTax, result.VenueTypeTax, "venuetypetax")
		assert.NotSame(t, given.Publisher, result.Publisher, "publisher")
		assert.NotSame(t, given.Publisher.Ext, result.Publisher.Ext, "publisher-ext")
		assert.NotSame(t, given.Content, result.Content, "content")
		assert.NotSame(t, given.Content.Ext, result.Content.Ext, "content-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.DOOH{})),
			[]string{
				"VenueType",
				"VenueTypeTax",
				"Publisher",
				"Content",
				"Ext",
			})
	})
}

func TestCloneBidderReq(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneBidderReq(nil)
		assert.Nil(t, result)
	})

	t.Run("empty", func(t *testing.T) {
		given := &openrtb2.BidRequest{}
		result := CloneBidderReq(given)
		assert.Equal(t, given, result.BidRequest)
		assert.NotSame(t, given, result)
	})

	t.Run("populated", func(t *testing.T) {
		given := &openrtb2.BidRequest{
			ID:     "anyID",
			User:   &openrtb2.User{ID: "testUserId"},
			Device: &openrtb2.Device{Carrier: "testCarrier"},
			Source: &openrtb2.Source{TID: "testTID"},
		}
		result := CloneBidderReq(given)
		assert.Equal(t, given, result.BidRequest)
		assert.NotSame(t, given, result.BidRequest, "pointer")
		assert.NotSame(t, given.User, result.User, "user")
		assert.NotSame(t, given.Device, result.Device, "device")
		assert.NotSame(t, given.Source, result.Source, "source")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverPointerFields(reflect.TypeOf(openrtb2.BidRequest{})),
			[]string{
				"Device",
				"User",
				"Source",
				"Imp",
				"Site",
				"App",
				"DOOH",
				"WSeat",
				"BSeat",
				"Cur",
				"WLang",
				"WLangB",
				"BCat",
				"BAdv",
				"BApp",
				"Regs",
				"Ext",
			})
	})
}

// discoverPointerFields returns the names of all fields of an object that are
// pointers and would need to be cloned. This method is specific to types which can
// appear within an OpenRTB data model object.
func discoverPointerFields(t reflect.Type) []string {
	var fields []string
	for _, f := range reflect.VisibleFields(t) {
		if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Pointer {
			fields = append(fields, f.Name)
		}
	}
	return fields
}
