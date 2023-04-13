package ortb

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v17/adcom1"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneApp(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		result := CloneApp(nil)
		assert.Nil(t, result)
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.App{})),
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Publisher{})),
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
		assert.NotSame(t, given.Network, result.Network, "network")
		assert.NotSame(t, given.Network.Ext, result.Network.Ext, "network-ext")
		assert.NotSame(t, given.Channel, result.Channel, "channel")
		assert.NotSame(t, given.Channel.Ext, result.Channel.Ext, "channel-ext")
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Content{})),
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Producer{})),
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
		given := []openrtb2.Segment{}
		result := CloneSegmentSlice(given)
		assert.Empty(t, result)
		assert.NotSame(t, given, result)
	})

	t.Run("one", func(t *testing.T) {
		given := []openrtb2.Data{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
		}
		result := CloneDataSlice(given)
		require.Len(t, result, 1)
		assert.NotSame(t, given, result)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.Segment{
			{ID: "1", Ext: json.RawMessage(`{"anyField":1}`)},
			{ID: "2", Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneSegmentSlice(given)
		require.Len(t, result, 2)
		assert.NotSame(t, given, result)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
		assert.NotSame(t, given[0].Ext, result[0].Ext)
		assert.Equal(t, given[1], result[1])
		assert.NotSame(t, given[1], result[1])
		assert.NotSame(t, given[1].Ext, result[1].Ext)
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
		assert.NotSame(t, given.Ext, result.Ext, "ext")
	})

	t.Run("assumptions", func(t *testing.T) {
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Data{})),
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
		require.Len(t, result, 1)
		assert.NotSame(t, given, result)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
	})

	t.Run("many", func(t *testing.T) {
		given := []openrtb2.Segment{
			{Ext: json.RawMessage(`{"anyField":1}`)},
			{Ext: json.RawMessage(`{"anyField":2}`)},
		}
		result := CloneSegmentSlice(given)
		require.Len(t, result, 2)
		assert.NotSame(t, given, result)
		assert.Equal(t, given[0], result[0])
		assert.NotSame(t, given[0], result[0])
		assert.NotSame(t, given[0].Ext, result[0].Ext)
		assert.Equal(t, given[1], result[1])
		assert.NotSame(t, given[1], result[1])
		assert.NotSame(t, given[1].Ext, result[1].Ext)
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Segment{})),
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Network{})),
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
		assert.ElementsMatch(t, discoverDirectPointerFields(reflect.TypeOf(openrtb2.Channel{})),
			[]string{
				"Ext",
			})
	})
}

// discoverDirectPointerFields returns the names of all fields of an object that are
// pointers and would need to be cloned. This method is specific to types which can
// appear within an OpenRTB data model object.
func discoverDirectPointerFields(t reflect.Type) []string {
	var fields []string
	for _, f := range reflect.VisibleFields(t) {
		if f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Pointer {
			fields = append(fields, f.Name)
		}
	}
	return fields
}
