package rtd

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"plain domain", "coursera.com", "coursera.com"},
		{"uppercase is lowered", "Coursera.COM", "coursera.com"},
		{"surrounding whitespace", "  coursera.com  ", "coursera.com"},
		{"www prefix stripped", "www.coursera.com", "coursera.com"},
		{"mobile prefix stripped", "m.coursera.com", "coursera.com"},
		{"amp prefix stripped", "amp.coursera.com", "coursera.com"},
		{"amp prefix stripped from url", "https://amp.coursera.com/learn", "coursera.com"},
		{"amp.dev is not reduced to dev", "amp.dev", "amp.dev"},
		{"m.co is not reduced to co", "m.co", "m.co"},
		{"www2 is not a variant prefix", "www2.coursera.com", "www2.coursera.com"},
		{"other subdomains preserved", "shop.coursera.com", "shop.coursera.com"},
		{"prefix only stripped at the front", "coursera.m.com", "coursera.m.com"},
		{"https url", "https://www.coursera.com/learn/python?ref=nav", "coursera.com"},
		{"http url with port", "http://coursera.com:8080/learn", "coursera.com"},
		{"bare host with path", "coursera.com/learn/python", "coursera.com"},
		{"bare host with port", "coursera.com:443", "coursera.com"},
		{"trailing dot", "coursera.com.", "coursera.com"},
		{"subdomain preserved", "blog.coursera.com", "blog.coursera.com"},
		{"android bundle kept", "com.example.app", "com.example.app"},
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"localhost rejected", "localhost", ""},
		{"single label rejected", "intranet", ""},
		{"ios store id rejected", "123456789", ""},
		{"ipv4 rejected", "192.168.1.1", ""},
		{"url with no host", "https:///learn", ""},
		{"malformed url", "https://exa mple.com/x", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, normalizeDomain(test.raw))
		})
	}
}

// TestDomainVariantsShareCacheKey pins the property the cache depends on:
// every spelling of the same site must normalize to one key, otherwise each
// variant would trigger its own classification call and its own cache entry.
func TestDomainVariantsShareCacheKey(t *testing.T) {
	variants := []string{
		"coursera.com",
		"www.coursera.com",
		"WWw.coursera.com",
		"COURSERA.COM",
		"m.coursera.com",
		"amp.coursera.com",
		"https://m.coursera.com/learn/python",
		"https://amp.coursera.com/learn",
		"AMP.Coursera.com",
		"coursera.com/xyz",
		"www.coursera.com/xyz",
		"https://www.coursera.com",
		"https://www.coursera.com/",
		"https://coursera.com/learn/python?ref=nav#top",
		"HTTPS://WWW.Coursera.COM/XYZ",
		"http://coursera.com:8080/learn",
		"coursera.com:443",
		"coursera.com.",
		"  coursera.com  ",
	}

	module := newTestModule(t, "https://example.com/v1/responses", nil)
	want := string(module.cacheKey("coursera.com"))

	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			assert.Equal(t, "coursera.com", normalizeDomain(variant))
			assert.Equal(t, want, string(module.cacheKey(normalizeDomain(variant))),
				"%q must share a cache entry with coursera.com", variant)
		})
	}
}

// TestRequestVariantsShareCacheKey covers the same property end to end: the
// domain may arrive on any of several ORTB fields, in any spelling.
func TestRequestVariantsShareCacheKey(t *testing.T) {
	requests := map[string]*openrtb2.BidRequest{
		"site.domain":           {Site: &openrtb2.Site{Domain: "coursera.com"}},
		"site.domain with www":  {Site: &openrtb2.Site{Domain: "WWW.Coursera.com"}},
		"site.page url":         {Site: &openrtb2.Site{Page: "https://www.coursera.com/learn/python"}},
		"site.page bare host":   {Site: &openrtb2.Site{Page: "coursera.com/learn"}},
		"site.publisher.domain": {Site: &openrtb2.Site{Publisher: &openrtb2.Publisher{Domain: "coursera.com."}}},
		"app.domain":            {App: &openrtb2.App{Domain: "https://coursera.com"}},
		"dooh.domain":           {DOOH: &openrtb2.DOOH{Domain: "www.coursera.com:8080"}},
	}

	module := newTestModule(t, "https://example.com/v1/responses", nil)
	want := string(module.cacheKey("coursera.com"))

	for name, request := range requests {
		t.Run(name, func(t *testing.T) {
			domain := resolveDomain(request)
			assert.Equal(t, "coursera.com", domain)
			assert.Equal(t, want, string(module.cacheKey(domain)))
		})
	}
}

func TestResolveDomain(t *testing.T) {
	tests := []struct {
		name    string
		request *openrtb2.BidRequest
		want    string
	}{
		{"nil request", nil, ""},
		{"empty request", &openrtb2.BidRequest{}, ""},
		{
			name:    "site domain wins",
			request: &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com", Page: "https://other.com/x"}},
			want:    "coursera.com",
		},
		{
			name:    "falls back to site page",
			request: &openrtb2.BidRequest{Site: &openrtb2.Site{Page: "https://www.coursera.com/learn/python"}},
			want:    "coursera.com",
		},
		{
			name: "falls back to site publisher domain",
			request: &openrtb2.BidRequest{Site: &openrtb2.Site{
				Publisher: &openrtb2.Publisher{Domain: "publisher.com"},
			}},
			want: "publisher.com",
		},
		{
			name: "skips unusable site page",
			request: &openrtb2.BidRequest{Site: &openrtb2.Site{
				Page:      "http://localhost:8080/test",
				Publisher: &openrtb2.Publisher{Domain: "publisher.com"},
			}},
			want: "publisher.com",
		},
		{
			name:    "app domain",
			request: &openrtb2.BidRequest{App: &openrtb2.App{Domain: "example.com", Bundle: "com.example.app"}},
			want:    "example.com",
		},
		{
			name:    "app bundle fallback",
			request: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "com.example.app"}},
			want:    "com.example.app",
		},
		{
			name: "app skips numeric ios bundle",
			request: &openrtb2.BidRequest{App: &openrtb2.App{
				Bundle:    "123456789",
				Publisher: &openrtb2.Publisher{Domain: "publisher.com"},
			}},
			want: "publisher.com",
		},
		{
			name:    "dooh domain",
			request: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{Domain: "screens.example.com"}},
			want:    "screens.example.com",
		},
		{
			name: "dooh publisher fallback",
			request: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{
				Publisher: &openrtb2.Publisher{Domain: "publisher.com"},
			}},
			want: "publisher.com",
		},
		{
			name: "site takes precedence over app",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{Domain: "site.com"},
				App:  &openrtb2.App{Domain: "app.com"},
			},
			want: "site.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, resolveDomain(test.request))
		})
	}
}

func TestEnrichWritesSegments(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	segs := segments{Content22: []string{"132", "148"}}

	request := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}
	assert.True(t, module.enrich(request, segs))

	require.NotNil(t, request.Site.Content)
	require.Len(t, request.Site.Content.Data, 1)

	data := request.Site.Content.Data[0]
	assert.Equal(t, DefaultDataProviderName, data.Name)
	assert.JSONEq(t, `{"segtax":6}`, string(data.Ext))
	assert.Equal(t, []openrtb2.Segment{{ID: "132"}, {ID: "148"}}, data.Segment)
}

func TestEnrichAllTaxonomies(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	segs := segments{
		Content22: []string{"132"},
		Content10: []string{"IAB5"},
		Audience:  []string{"23"},
	}

	request := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}
	assert.True(t, module.enrich(request, segs))

	require.Len(t, request.Site.Content.Data, 2)
	assert.JSONEq(t, `{"segtax":6}`, string(request.Site.Content.Data[0].Ext))
	assert.JSONEq(t, `{"segtax":1}`, string(request.Site.Content.Data[1].Ext))
	assert.Equal(t, []openrtb2.Segment{{ID: "IAB5"}}, request.Site.Content.Data[1].Segment)

	require.NotNil(t, request.User)
	require.Len(t, request.User.Data, 1)
	assert.JSONEq(t, `{"segtax":4}`, string(request.User.Data[0].Ext))
	assert.Equal(t, []openrtb2.Segment{{ID: "23"}}, request.User.Data[0].Segment)
}

func TestEnrichPreservesExistingData(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)

	request := &openrtb2.BidRequest{Site: &openrtb2.Site{
		Domain: "coursera.com",
		Content: &openrtb2.Content{
			Title: "existing content",
			Data: []openrtb2.Data{{
				Name:    "other-provider.com",
				Ext:     []byte(`{"segtax":6}`),
				Segment: []openrtb2.Segment{{ID: "999"}},
			}},
		},
	}}

	assert.True(t, module.enrich(request, segments{Content22: []string{"132"}}))

	assert.Equal(t, "existing content", request.Site.Content.Title)
	require.Len(t, request.Site.Content.Data, 2)
	assert.Equal(t, "other-provider.com", request.Site.Content.Data[0].Name)
	assert.Equal(t, DefaultDataProviderName, request.Site.Content.Data[1].Name)
}

func TestEnrichIsIdempotent(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	segs := segments{Content22: []string{"132"}, Audience: []string{"23"}}
	request := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}

	assert.True(t, module.enrich(request, segs))
	assert.False(t, module.enrich(request, segs), "a second pass must not append duplicates")

	assert.Len(t, request.Site.Content.Data, 1)
	assert.Len(t, request.User.Data, 1)
}

func TestEnrichAppAndDooh(t *testing.T) {
	tests := []struct {
		name    string
		request *openrtb2.BidRequest
		content func(*openrtb2.BidRequest) *openrtb2.Content
		wantKey []string
	}{
		{
			name:    "app",
			request: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "com.example.app"}},
			content: func(r *openrtb2.BidRequest) *openrtb2.Content { return r.App.Content },
			wantKey: []string{"bidrequest", "app", "content", "data"},
		},
		{
			name:    "dooh",
			request: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{Domain: "screens.example.com"}},
			content: func(r *openrtb2.BidRequest) *openrtb2.Content { return r.DOOH.Content },
			wantKey: []string{"bidrequest", "dooh", "content", "data"},
		},
	}

	module := newTestModule(t, "https://example.com/v1/responses", nil)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.True(t, module.enrich(test.request, segments{Content22: []string{"132"}}))

			content := test.content(test.request)
			require.NotNil(t, content)
			require.Len(t, content.Data, 1)
			assert.JSONEq(t, `{"segtax":6}`, string(content.Data[0].Ext))
			assert.Equal(t, test.wantKey, mutationKey(test.request))
		})
	}
}

func TestEnrichNoOpCases(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)

	t.Run("nil request", func(t *testing.T) {
		assert.False(t, module.enrich(nil, segments{Content22: []string{"132"}}))
	})

	t.Run("empty segments leave request untouched", func(t *testing.T) {
		request := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}
		assert.False(t, module.enrich(request, segments{}))
		assert.Nil(t, request.Site.Content, "no content object should be created")
		assert.Nil(t, request.User)
	})

	t.Run("no distribution channel", func(t *testing.T) {
		request := &openrtb2.BidRequest{}
		assert.False(t, module.enrich(request, segments{Content22: []string{"132"}}))
		assert.Equal(t, []string{"bidrequest"}, mutationKey(request))
	})

	t.Run("audience only does not create content", func(t *testing.T) {
		request := &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}
		assert.True(t, module.enrich(request, segments{Audience: []string{"23"}}))
		assert.Nil(t, request.Site.Content)
		assert.Len(t, request.User.Data, 1)
	})

	t.Run("existing user object is reused", func(t *testing.T) {
		request := &openrtb2.BidRequest{
			Site: &openrtb2.Site{Domain: "coursera.com"},
			User: &openrtb2.User{ID: "user-1"},
		}
		assert.True(t, module.enrich(request, segments{Audience: []string{"23"}}))
		assert.Equal(t, "user-1", request.User.ID)
		assert.Len(t, request.User.Data, 1)
	})
}

func TestHasDataEntry(t *testing.T) {
	tests := []struct {
		name   string
		data   []openrtb2.Data
		segtax int
		want   bool
	}{
		{"empty", nil, segtaxContent22, false},
		{
			name:   "matching name and segtax",
			data:   []openrtb2.Data{{Name: DefaultDataProviderName, Ext: []byte(`{"segtax":6}`)}},
			segtax: segtaxContent22,
			want:   true,
		},
		{
			name:   "same name different segtax",
			data:   []openrtb2.Data{{Name: DefaultDataProviderName, Ext: []byte(`{"segtax":1}`)}},
			segtax: segtaxContent22,
			want:   false,
		},
		{
			name:   "different provider",
			data:   []openrtb2.Data{{Name: "other.com", Ext: []byte(`{"segtax":6}`)}},
			segtax: segtaxContent22,
			want:   false,
		},
		{
			name:   "no ext",
			data:   []openrtb2.Data{{Name: DefaultDataProviderName}},
			segtax: segtaxContent22,
			want:   false,
		},
		{
			name:   "unparseable ext is ignored",
			data:   []openrtb2.Data{{Name: DefaultDataProviderName, Ext: []byte(`not json`)}},
			segtax: segtaxContent22,
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, hasDataEntry(test.data, DefaultDataProviderName, test.segtax))
		})
	}
}

func TestBuildDataRejectsEmptyIDs(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	_, ok := module.buildData(nil, nil, segtaxContent22)
	assert.False(t, ok)
}

func TestIsAllDigitsAndDots(t *testing.T) {
	assert.True(t, isAllDigitsAndDots("192.168.1.1"))
	assert.True(t, isAllDigitsAndDots("123"))
	assert.False(t, isAllDigitsAndDots("com.example.app"))
	assert.False(t, isAllDigitsAndDots("a1.com"))
}
