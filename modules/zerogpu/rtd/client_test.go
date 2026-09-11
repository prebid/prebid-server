package rtd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coocood/freecache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// classificationJSON is the inner payload returned by the domain classifier,
// taken verbatim from the ZeroGPU API documentation for coursera.com.
const classificationJSON = `{"audience":[{"id":23,"parent_id":20,"name":"Undergraduate Education","tier1_name":"Demographic","score":0.86137356782421},{"id":20,"parent_id":17,"name":"College Education","tier1_name":"Demographic","score":0.8185467622936815}],"content":{"iab_1_0":[{"code":"IAB5","name":"Education","tier":1,"parent_code":null,"score":0.9975345244047042},{"code":"IAB5-6","name":"Distance Learning","tier":2,"parent_code":"IAB5","score":0.9304775059727456}],"iab_2_2":[{"id":132,"parent_id":0,"name":"Education","tier1_name":"Education","score":0.9975345244047042},{"id":148,"parent_id":132,"name":"Online Education","tier1_name":"Education","tier2_name":"Online Education","score":0.9304775059727456}]}}`

// responsesEnvelope wraps a classification in a /v1/responses envelope.
func responsesEnvelope(payload string) string {
	quoted, _ := json.Marshal(payload)
	return `{"id":"c961f004","object":"response","status":"completed","model":"` + DefaultModel +
		`","output":[{"type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":` +
		string(quoted) + `,"annotations":[]}]}]}`
}

// newTestModule builds a module wired to the given server URL. Background
// warm-ups are drained on cleanup so no goroutine outlives the test server.
func newTestModule(t *testing.T, endpoint string, mutate func(*Config)) *Module {
	t.Helper()

	cfg := Config{APIKey: "test-key", Endpoint: endpoint}
	cfg.applyDefaults()
	if mutate != nil {
		mutate(&cfg)
	}
	require.NoError(t, cfg.validate())

	bgCtx, bgCancel := context.WithCancel(context.Background())

	module := &Module{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Millisecond},
		cache:      freecache.NewCache(cfg.CacheSize),
		bgCtx:      bgCtx,
		bgCancel:   bgCancel,
	}

	// Shutdown waits for in-flight warm-ups, so the test server outlives them.
	t.Cleanup(func() { _ = module.Shutdown() })

	return module
}

// awaitWarmUps blocks until every in-flight warm-up has finished, using the
// same WaitGroup Shutdown waits on.
func awaitWarmUps(m *Module) { m.wg.Wait() }

// primeCache warms a domain and waits for it to land in the cache.
func primeCache(t *testing.T, module *Module, domain string) {
	t.Helper()
	module.warm(domain)
	awaitWarmUps(module)

	if _, cached := module.lookup(domain); !cached {
		t.Fatalf("warm-up did not populate the cache for %q", domain)
	}
}

func TestFetchSendsResponsesRequest(t *testing.T) {
	var gotBody map[string]interface{}
	var gotAPIKey, gotContentType, gotMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAPIKey = r.Header.Get("x-api-key")
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responsesEnvelope(classificationJSON)))
	}))
	defer server.Close()

	module := newTestModule(t, server.URL+"/v1/responses", nil)
	segs, err := module.fetch(context.Background(), "coursera.com")
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "test-key", gotAPIKey)
	assert.Equal(t, "application/json", gotContentType)

	// The Responses API takes a bare `input` string.
	assert.Equal(t, DefaultModel, gotBody["model"])
	assert.Equal(t, "coursera.com", gotBody["input"])
	assert.Len(t, gotBody, 2, "only model and input should be sent")

	assert.Equal(t, []string{"132", "148"}, segs.Content22)
	assert.Empty(t, segs.Content10, "content 1.0 is opt-in")
	assert.Empty(t, segs.Audience, "audience is opt-in")
}

func TestFetchEnrichmentFlags(t *testing.T) {
	tests := []struct {
		name          string
		mutate        func(*Config)
		wantContent22 []string
		wantContent10 []string
		wantAudience  []string
	}{
		{
			name:          "defaults emit content 2.2 only",
			mutate:        nil,
			wantContent22: []string{"132", "148"},
		},
		{
			name:          "content 1.0 enabled",
			mutate:        func(c *Config) { c.EnrichContent10 = true },
			wantContent22: []string{"132", "148"},
			wantContent10: []string{"IAB5", "IAB5-6"},
		},
		{
			name:          "audience enabled",
			mutate:        func(c *Config) { c.EnrichUserAudience = true },
			wantContent22: []string{"132", "148"},
			wantAudience:  []string{"23", "20"},
		},
		{
			name: "all taxonomies enabled",
			mutate: func(c *Config) {
				c.EnrichContent10 = true
				c.EnrichUserAudience = true
			},
			wantContent22: []string{"132", "148"},
			wantContent10: []string{"IAB5", "IAB5-6"},
			wantAudience:  []string{"23", "20"},
		},
		{
			name:          "min_score filters low confidence",
			mutate:        func(c *Config) { c.MinScore = 0.95; c.EnrichContent10 = true; c.EnrichUserAudience = true },
			wantContent22: []string{"132"},
			wantContent10: []string{"IAB5"},
			wantAudience:  nil,
		},
		{
			name: "max_segments caps each taxonomy",
			mutate: func(c *Config) {
				c.MaxSegments = 1
				c.EnrichContent10 = true
				c.EnrichUserAudience = true
			},
			wantContent22: []string{"132"},
			wantContent10: []string{"IAB5"},
			wantAudience:  []string{"23"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
			defer server.Close()

			module := newTestModule(t, server.URL, test.mutate)
			segs, err := module.fetch(context.Background(), "coursera.com")
			require.NoError(t, err)

			assert.Equal(t, test.wantContent22, segs.Content22)
			assert.Equal(t, test.wantContent10, segs.Content10)
			assert.Equal(t, test.wantAudience, segs.Audience)
		})
	}
}

func TestFetchErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		wantErr       string
		wantTransient bool
	}{
		{"bad request is permanent", http.StatusBadRequest, ``, "status 400", false},
		{"unauthorized is permanent", http.StatusUnauthorized, ``, "status 401", false},
		{"forbidden is permanent", http.StatusForbidden, ``, "status 403", false},
		{"insufficient quota is permanent", statusInsufficientQuota, ``, "status 420", false},
		{"server error is transient", http.StatusInternalServerError, ``, "status 500", true},
		{"undocumented status is transient", http.StatusBadGateway, ``, "status 502", true},
		{"unparseable envelope", http.StatusOK, `not json`, "failed to decode", false},
		{"empty envelope", http.StatusOK, `{"output":[]}`, "no classification payload", false},
		{"envelope with blank text", http.StatusOK, `{"output":[{"content":[{"text":""}]}]}`, "no classification payload", false},
		{"unparseable classification", http.StatusOK, responsesEnvelope(`{"content":`), "failed to parse classification", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newClassifierServer(t, test.status, test.body)
			defer server.Close()

			module := newTestModule(t, server.URL, nil)
			_, err := module.fetch(context.Background(), "coursera.com")

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
			assert.Equal(t, test.wantTransient, isTransient(err))
		})
	}
}

func TestFetchTransportFailureIsTransient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	endpoint := server.URL
	server.Close() // nothing is listening now

	module := newTestModule(t, endpoint, nil)
	_, err := module.fetch(context.Background(), "coursera.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "request to ZeroGPU failed")
	assert.True(t, isTransient(err))
}

func TestFetchTimeoutIsTransient(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	module := newTestModule(t, server.URL, func(c *Config) { c.Timeout = 20 })
	_, err := module.fetch(context.Background(), "coursera.com")

	require.Error(t, err)
	assert.True(t, isTransient(err))
}

func TestFetchInvalidEndpointBuildFailure(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	// A control character makes http.NewRequestWithContext fail.
	module.cfg.Endpoint = "https://example.com/\x7f"

	_, err := module.fetch(context.Background(), "coursera.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build request")
	assert.False(t, isTransient(err))
}

func TestLookupIsCacheOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("lookup must never perform I/O")
	}))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	segs, cached := module.lookup("coursera.com")
	assert.False(t, cached, "an unseen domain is not cached")
	assert.True(t, segs.isEmpty())
}

func TestWarmPopulatesCache(t *testing.T) {
	var mu sync.Mutex
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		_, _ = w.Write([]byte(responsesEnvelope(classificationJSON)))
	}))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	// Before warming, the domain is unknown.
	_, cached := module.lookup("coursera.com")
	require.False(t, cached)

	primeCache(t, module, "coursera.com")

	segs, cached := module.lookup("coursera.com")
	assert.True(t, cached)
	assert.Equal(t, []string{"132", "148"}, segs.Content22)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestWarmCollapsesConcurrentRequestsForSameDomain(t *testing.T) {
	var mu sync.Mutex
	var calls int
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release // hold the request open so every warm() lands while one is in flight
		_, _ = w.Write([]byte(responsesEnvelope(classificationJSON)))
	}))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	// Fire many warm-ups for the same domain while the first is still running.
	for i := 0; i < 25; i++ {
		module.warm("coursera.com")
	}
	close(release)
	awaitWarmUps(module)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "concurrent warm-ups for one domain must collapse to a single request")
}

func TestWarmTracksDomainsIndependently(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	primeCache(t, module, "coursera.com")
	primeCache(t, module, "example.com")

	_, cached := module.lookup("coursera.com")
	assert.True(t, cached)
	_, cached = module.lookup("example.com")
	assert.True(t, cached)
	_, cached = module.lookup("unseen.com")
	assert.False(t, cached)
}

func TestWarmReleasesDomainAfterCompletion(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	primeCache(t, module, "coursera.com")

	// The in-flight marker must be cleared, otherwise a domain could never be
	// re-warmed after its cache entry expires.
	_, stillMarked := module.inFlight.Load("coursera.com")
	assert.False(t, stillMarked)
}

func TestWarmNegativeCacheSuppressesRepeatCalls(t *testing.T) {
	var mu sync.Mutex
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)

	module.warm("coursera.com")
	awaitWarmUps(module)

	// The failure is cached, so lookup reports a hit carrying no segments.
	segs, cached := module.lookup("coursera.com")
	assert.True(t, cached, "a suppressed failure is a cache hit")
	assert.True(t, segs.isEmpty())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestShutdownStopsWarmUps(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	primeCache(t, module, "coursera.com")

	require.NoError(t, module.Shutdown())
	require.NoError(t, module.Shutdown(), "shutdown must be idempotent")

	// After shutdown no new warm-up is registered at all, so nothing races
	// against the WaitGroup the host is already waiting on.
	module.warm("example.com")
	awaitWarmUps(module)

	_, cached := module.lookup("example.com")
	assert.False(t, cached, "no warm-up should have run after shutdown")
}

func TestCacheNegativeTTLSelection(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantTTL int
	}{
		{"transient uses retry ttl", transientErrorf("boom"), defaultRetryCacheTTLSeconds},
		{"permanent uses negative ttl", permanentErrorf("boom"), defaultNegativeCacheTTLSeconds},
		{"unknown error treated as transient", errors.New("boom"), defaultRetryCacheTTLSeconds},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			module := newTestModule(t, "https://example.com/v1/responses", nil)
			key := module.cacheKey("coursera.com")

			module.cacheNegative(key, test.err)

			value, err := module.cache.Get(key)
			require.NoError(t, err)
			assert.Equal(t, "{}", string(value))

			ttl, err := module.cache.TTL(key)
			require.NoError(t, err)
			assert.InDelta(t, test.wantTTL, int(ttl), 2)
		})
	}
}

func TestCacheNegativeSkippedWhenTTLZero(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", func(c *Config) {
		c.RetryCacheTTLSeconds = 0
	})
	// applyDefaults already ran, so force the zero explicitly.
	module.cfg.RetryCacheTTLSeconds = 0

	key := module.cacheKey("coursera.com")
	module.cacheNegative(key, transientErrorf("boom"))

	_, err := module.cache.Get(key)
	assert.Error(t, err, "nothing should have been cached")
}

func TestCacheResultUsesNegativeTTLForEmptyResult(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	key := module.cacheKey("coursera.com")

	module.cacheResult(key, segments{})
	ttl, err := module.cache.TTL(key)
	require.NoError(t, err)
	assert.InDelta(t, defaultNegativeCacheTTLSeconds, int(ttl), 2)

	module.cacheResult(key, segments{Content22: []string{"132"}})
	ttl, err = module.cache.TTL(key)
	require.NoError(t, err)
	assert.InDelta(t, defaultCacheTTLSeconds, int(ttl), 2)
}

func TestLookupDiscardsCorruptCacheEntry(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	key := module.cacheKey("coursera.com")
	require.NoError(t, module.cache.Set(key, []byte("not json"), 60))

	// A corrupt entry must read as a miss and be evicted, so the domain can be
	// re-warmed rather than staying permanently broken.
	_, cached := module.lookup("coursera.com")
	assert.False(t, cached)
	_, err := module.cache.Get(key)
	assert.Error(t, err, "the corrupt entry should have been deleted")

	primeCache(t, module, "coursera.com")
	segs, cached := module.lookup("coursera.com")
	assert.True(t, cached)
	assert.Equal(t, []string{"132", "148"}, segs.Content22)
}

func TestWarmEmptyResultIsNotAnError(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(`{"content":{"iab_2_2":[]}}`))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	module.warm("coursera.com")
	awaitWarmUps(module)

	// A classification with no qualifying categories is still cached, so the
	// domain is not re-queried on every auction.
	segs, cached := module.lookup("coursera.com")
	assert.True(t, cached)
	assert.True(t, segs.isEmpty())
}

func TestFilterSkipsZeroIdentifiers(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", func(c *Config) {
		c.EnrichContent10 = true
		c.EnrichUserAudience = true
	})

	result := classification{
		Audience: []scoredID{{ID: 0, Score: 0.99}, {ID: 20, Score: 0.99}},
		Content: contentByTaxon{
			IAB10: []scoredCode{{Code: "", Score: 0.99}, {Code: "IAB5", Score: 0.99}},
			IAB22: []scoredID{{ID: 0, Score: 0.99}, {ID: 132, Score: 0.99}},
		},
	}

	segs := module.filter(result)
	assert.Equal(t, []string{"132"}, segs.Content22)
	assert.Equal(t, []string{"IAB5"}, segs.Content10)
	assert.Equal(t, []string{"20"}, segs.Audience)
}

func TestSegmentsIsEmpty(t *testing.T) {
	assert.True(t, segments{}.isEmpty())
	assert.False(t, segments{Content22: []string{"1"}}.isEmpty())
	assert.False(t, segments{Content10: []string{"IAB5"}}.isEmpty())
	assert.False(t, segments{Audience: []string{"1"}}.isEmpty())
}

func TestApiEnvelopePayload(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"real envelope", responsesEnvelope("the-payload"), "the-payload"},
		{"first non-empty text wins", `{"output":[{"content":[{"text":""},{"text":"second"}]}]}`, "second"},
		{"skips empty message", `{"output":[{"content":[]},{"content":[{"text":"later"}]}]}`, "later"},
		{"no output", `{"output":[]}`, ""},
		{"absent output", `{}`, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var envelope apiEnvelope
			require.NoError(t, json.Unmarshal([]byte(test.raw), &envelope))
			assert.Equal(t, test.want, envelope.payload())
		})
	}

	assert.Empty(t, apiEnvelope{}.payload())
}

func TestCacheKeyVariesByModel(t *testing.T) {
	module := newTestModule(t, "https://example.com/v1/responses", nil)
	other := newTestModule(t, "https://example.com/v1/responses", func(c *Config) { c.Model = "other-model" })

	assert.NotEqual(t, string(module.cacheKey("a.com")), string(other.cacheKey("a.com")))
	assert.NotEqual(t, string(module.cacheKey("a.com")), string(module.cacheKey("b.com")))
}

// newClassifierServer returns a server replying with a fixed status and body.
func newClassifierServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}
