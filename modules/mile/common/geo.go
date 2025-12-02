package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// GeoResolver resolves country codes from IP addresses.
type GeoResolver interface {
	Resolve(ctx context.Context, ip string) (string, error)
}

// geoCacheEntry represents a cached geo lookup result.
type geoCacheEntry struct {
	country string
	expires time.Time
}

// HTTPGeoResolver resolves geolocation via HTTP endpoint with in-memory cache.
type HTTPGeoResolver struct {
	endpoint string
	ttl      time.Duration
	client   *http.Client
	cache    sync.Map // map[string]*geoCacheEntry
}

// NewHTTPGeoResolver creates a new HTTP-based GeoResolver.
func NewHTTPGeoResolver(endpoint string, ttl time.Duration, client *http.Client) (*HTTPGeoResolver, error) {
	if endpoint == "" {
		return nil, errors.New("geo endpoint required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPGeoResolver{
		endpoint: endpoint,
		ttl:      ttl,
		client:   client,
	}, nil
}

// Resolve returns ISO alpha-2 country code for the given IP. It caches successful lookups.
func (r *HTTPGeoResolver) Resolve(ctx context.Context, ip string) (string, error) {
	if ip == "" {
		return "", errors.New("ip required")
	}
	normalizedIP := normalizeIP(ip)
	if normalizedIP == "" {
		return "", fmt.Errorf("invalid ip: %s", ip)
	}

	// Check cache
	if cached, ok := r.cache.Load(normalizedIP); ok {
		entry := cached.(*geoCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.country, nil
		}
		r.cache.Delete(normalizedIP)
	}

	country, err := r.fetchCountry(ctx, normalizedIP)
	if err != nil {
		return "", err
	}

	r.cache.Store(normalizedIP, &geoCacheEntry{
		country: country,
		expires: time.Now().Add(r.ttl),
	})

	return country, nil
}

func (r *HTTPGeoResolver) fetchCountry(ctx context.Context, ip string) (string, error) {
	url := strings.ReplaceAll(r.endpoint, "{ip}", ip)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("geo fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("geo endpoint status %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode geo response: %w", err)
	}

	if country, ok := extractCountryFromPayload(payload); ok {
		return country, nil
	}

	return "", errors.New("country not found in geo response")
}

func normalizeIP(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	// Convert IPv6 zones (e.g., "fe80::1%lo0") to plain address
	if ipv6 := parsed.To16(); ipv6 != nil {
		return parsed.String()
	}
	return parsed.String()
}

func extractCountryFromPayload(payload map[string]any) (string, bool) {
	// Common field names: country, countryCode, country_code, iso_code
	keys := []string{"country", "countryCode", "country_code", "iso_code", "isoCode"}
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if country, ok := value.(string); ok && len(country) >= 2 {
				return strings.ToUpper(country[:2]), true
			}
		}
	}
	// Nested structure: { location: { country: "US" } }
	if location, ok := payload["location"].(map[string]any); ok {
		return extractCountryFromPayload(location)
	}
	return "", false
}

// MaxMindGeoResolver resolves geolocation using MaxMind GeoIP2 database
type MaxMindGeoResolver struct {
	db *geoip2.Reader
}

// NewMaxMindGeoResolver creates a new MaxMind-based GeoResolver
func NewMaxMindGeoResolver(dbPath string) (*MaxMindGeoResolver, error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MaxMind database: %w", err)
	}
	return &MaxMindGeoResolver{db: db}, nil
}

// Resolve returns ISO alpha-2 country code for the given IP
func (r *MaxMindGeoResolver) Resolve(_ context.Context, ip string) (string, error) {
	if ip == "" {
		return "", errors.New("ip required")
	}
	if r.db == nil {
		return "", errors.New("maxmind database not initialized")
	}
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return "", fmt.Errorf("invalid ip: %s", ip)
	}

	record, err := r.db.Country(parsedIP)
	if err != nil {
		return "", fmt.Errorf("maxmind lookup failed: %w", err)
	}

	if record.Country.IsoCode == "" {
		return "", errors.New("country code not found")
	}

	return strings.ToUpper(record.Country.IsoCode), nil
}

// Close closes the MaxMind database
func (r *MaxMindGeoResolver) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
