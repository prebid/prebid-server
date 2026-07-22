package doohqty

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prebid/openrtb/v20/adcom1"
)

const (
	csvSnapshotMaxBodyBytes = 10 * 1024 * 1024
	csvSnapshotMaxWarnings  = 20
)

type csvSnapshotSource struct {
	ctx       context.Context
	cancel    context.CancelFunc
	client    *http.Client
	mu        sync.Mutex
	snapshots map[string]*csvPublisherSnapshot
	wg        sync.WaitGroup
	closed    bool
}

type csvPublisherSnapshot struct {
	values      map[lookupKey]impressionValue
	warnings    []string
	lastErr     string
	lastSync    time.Time
	lastAttempt time.Time
	refreshing  bool
	hasSnapshot bool
}

func newCSVSnapshotSource(parent context.Context, client *http.Client) *csvSnapshotSource {
	ctx, cancel := context.WithCancel(parent)
	return &csvSnapshotSource{
		ctx:       ctx,
		cancel:    cancel,
		client:    client,
		snapshots: make(map[string]*csvPublisherSnapshot),
	}
}

func (s *csvSnapshotSource) Lookup(cfg moduleConfig, accountID string, lookups []lookupKey) (map[lookupKey]impressionValue, []string) {
	values := make(map[lookupKey]impressionValue, len(lookups))
	warnings := make([]string, 0)
	if s == nil {
		return values, []string{"DOOH qty CSV source is not initialized"}
	}

	cacheKey := csvSnapshotCacheKey(accountID, cfg.Source)
	now := time.Now()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return values, []string{"DOOH qty CSV source is shutting down"}
	}

	snapshot := s.snapshots[cacheKey]
	if snapshot == nil {
		snapshot = &csvPublisherSnapshot{}
		s.snapshots[cacheKey] = snapshot
	}

	if s.shouldRefreshLocked(snapshot, cfg.Source.SyncRateSeconds, now) {
		s.startRefreshLocked(cacheKey, snapshot, cfg, accountID, now)
	}

	hasSnapshot := snapshot.hasSnapshot
	snapshotValues := snapshot.values
	snapshotWarnings := append([]string(nil), snapshot.warnings...)
	lastErr := snapshot.lastErr
	refreshing := snapshot.refreshing
	s.mu.Unlock()

	if !hasSnapshot {
		if refreshing {
			warnings = append(warnings, fmt.Sprintf("DOOH qty CSV snapshot is loading for account %q", accountID))
		} else {
			warnings = append(warnings, fmt.Sprintf("DOOH qty CSV snapshot is not loaded for account %q", accountID))
		}
		if lastErr != "" {
			warnings = append(warnings, fmt.Sprintf("DOOH qty CSV refresh failed for account %q: %s", accountID, lastErr))
		}
		return values, warnings
	}

	if lastErr != "" {
		warnings = append(warnings, fmt.Sprintf("DOOH qty CSV refresh failed for account %q: %s; using last successful snapshot", accountID, lastErr))
	}
	warnings = append(warnings, snapshotWarnings...)

	for _, lookup := range lookups {
		value, ok := snapshotValues[lookup]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("no DOOH qty found for %s=%q", lookup.Path, lookup.Key))
			continue
		}
		values[lookup] = value
	}

	return values, warnings
}

func (s *csvSnapshotSource) shouldRefreshLocked(snapshot *csvPublisherSnapshot, syncRateSeconds int, now time.Time) bool {
	if snapshot.refreshing {
		return false
	}
	if !snapshot.hasSnapshot {
		return snapshot.lastAttempt.IsZero() || now.Sub(snapshot.lastAttempt) >= time.Duration(syncRateSeconds)*time.Second
	}
	return now.Sub(snapshot.lastSync) >= time.Duration(syncRateSeconds)*time.Second
}

func (s *csvSnapshotSource) startRefreshLocked(cacheKey string, snapshot *csvPublisherSnapshot, cfg moduleConfig, accountID string, now time.Time) {
	snapshot.refreshing = true
	snapshot.lastAttempt = now
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		values, warnings, err := s.fetchSnapshot(cfg, accountID)

		s.mu.Lock()
		defer s.mu.Unlock()

		snapshot := s.snapshots[cacheKey]
		if snapshot == nil {
			return
		}

		snapshot.refreshing = false
		if err != nil {
			snapshot.lastErr = err.Error()
			snapshot.warnings = warnings
			return
		}

		snapshot.values = values
		snapshot.warnings = warnings
		snapshot.lastErr = ""
		snapshot.lastSync = time.Now()
		snapshot.hasSnapshot = true
	}()
}

func (s *csvSnapshotSource) fetchSnapshot(cfg moduleConfig, accountID string) (map[lookupKey]impressionValue, []string, error) {
	requestCtx, cancel := context.WithTimeout(s.ctx, time.Duration(cfg.TimeoutMS)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, cfg.Source.Endpoint, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build CSV request: %s", err)
	}
	req.Header.Set("Accept", "text/csv")
	for name, value := range cfg.Source.Headers {
		req.Header.Set(name, value)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute CSV request: %s", err)
	}
	defer resp.Body.Close()

	body := io.LimitReader(resp.Body, csvSnapshotMaxBodyBytes)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, readErr := io.ReadAll(body)
		if readErr != nil {
			return nil, nil, fmt.Errorf("CSV endpoint returned status %d and response could not be read: %s", resp.StatusCode, readErr)
		}
		return nil, nil, fmt.Errorf("CSV endpoint returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	values, warnings, err := parseImpressionValueCSV(accountID, body)
	if err != nil {
		return nil, warnings, err
	}

	return values, warnings, nil
}

func (s *csvSnapshotSource) Shutdown() {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
}

func csvSnapshotCacheKey(accountID string, cfg sourceConfig) string {
	return accountID + "\x1f" + string(cfg.Type) + "\x1f" + cfg.Endpoint
}

type impressionValueCSVColumns struct {
	path       int
	key        int
	multiplier int
	sourceType int
	vendor     int
}

func parseImpressionValueCSV(accountID string, data io.Reader) (map[lookupKey]impressionValue, []string, error) {
	reader := csv.NewReader(data)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV header: %s", err)
	}

	columns, err := csvImpressionValueColumns(header)
	if err != nil {
		return nil, nil, err
	}

	values := make(map[lookupKey]impressionValue)
	warnings := make([]string, 0)
	line := 1
	for {
		line++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			warnings = appendCSVSnapshotWarning(warnings, fmt.Sprintf("CSV row %d skipped: %s", line, err))
			continue
		}

		value, ok, warning := impressionValueFromCSVRecord(columns, record)
		if warning != "" {
			warnings = appendCSVSnapshotWarning(warnings, fmt.Sprintf("CSV row %d skipped: %s", line, warning))
			continue
		}
		if !ok {
			continue
		}

		lookup := lookupKey{AccountID: accountID, Path: value.Path, Key: value.Key}
		if _, exists := values[lookup]; exists {
			warnings = appendCSVSnapshotWarning(warnings, fmt.Sprintf("CSV row %d skipped: duplicate value for %s=%q", line, value.Path, value.Key))
			continue
		}
		values[lookup] = value
	}

	return values, warnings, nil
}

func csvImpressionValueColumns(header []string) (impressionValueCSVColumns, error) {
	columns := impressionValueCSVColumns{
		path:       -1,
		key:        -1,
		multiplier: -1,
		sourceType: -1,
		vendor:     -1,
	}

	for index, name := range header {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "path":
			columns.path = index
		case "key":
			columns.key = index
		case "multiplier":
			columns.multiplier = index
		case "sourcetype":
			columns.sourceType = index
		case "vendor":
			columns.vendor = index
		}
	}

	if columns.path < 0 || columns.key < 0 || columns.multiplier < 0 {
		return columns, fmt.Errorf("CSV header must include path, key, and multiplier columns")
	}

	return columns, nil
}

func impressionValueFromCSVRecord(columns impressionValueCSVColumns, record []string) (impressionValue, bool, string) {
	path := strings.TrimSpace(csvRecordValue(record, columns.path))
	key := strings.TrimSpace(csvRecordValue(record, columns.key))
	if path == "" || key == "" {
		return impressionValue{}, false, "path or key is empty"
	}
	if !isSupportedLookupPath(path) {
		return impressionValue{}, false, fmt.Sprintf("lookup path %q is not supported", path)
	}

	multiplier, err := strconv.ParseFloat(strings.TrimSpace(csvRecordValue(record, columns.multiplier)), 64)
	if err != nil {
		return impressionValue{}, false, fmt.Sprintf("multiplier is invalid: %s", err)
	}

	sourceType := adcom1.MultiplierUnknown
	rawSourceType := strings.TrimSpace(csvRecordValue(record, columns.sourceType))
	if rawSourceType != "" {
		sourceTypeValue, err := strconv.Atoi(rawSourceType)
		if err != nil {
			return impressionValue{}, false, fmt.Sprintf("sourcetype is invalid: %s", err)
		}
		sourceType = adcom1.DOOHMultiplierMeasurementSourceType(sourceTypeValue)
	}

	value := impressionValue{
		Path:       path,
		Key:        key,
		Multiplier: multiplier,
		SourceType: sourceType,
		Vendor:     strings.TrimSpace(csvRecordValue(record, columns.vendor)),
	}
	if err := validateImpressionValue(value); err != nil {
		return impressionValue{}, false, err.Error()
	}

	return value, true, ""
}

func csvRecordValue(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return record[index]
}

func appendCSVSnapshotWarning(warnings []string, warning string) []string {
	if len(warnings) < csvSnapshotMaxWarnings {
		return append(warnings, warning)
	}
	if len(warnings) == csvSnapshotMaxWarnings {
		return append(warnings, "additional CSV warnings omitted")
	}
	return warnings
}
