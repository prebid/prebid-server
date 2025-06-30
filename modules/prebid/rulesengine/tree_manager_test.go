package rulesengine

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTreeManagerShutdown(t *testing.T) {
	tm := &treeManager{
		done:     make(chan struct{}),
		requests: make(chan buildInstruction, 10),
		monitor:  &treeManagerLogger{},
	}
	cache := NewCache()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := tm.Run(cache)
		wg.Done()
		assert.NoError(t, err)
	}()

	tm.Shutdown()

	wg.Wait()
	assert.Equal(t, int64(1), glog.Stats.Info.Lines())
}

func TestTreeManagerRun(t *testing.T) {
	t.Skip()
	schemaFile := "config/rules-engine-schema.json"
	validator, err := config.CreateSchemaValidator(schemaFile)
	assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", schemaFile))

	testCases := []struct {
		name                string
		inBuildInstruction  buildInstruction
		inStoredDataInCache map[accountID]*cacheEntry
		expectedCachedData  map[accountID]*cacheEntry
		expectedLogCalls    []string
		expectedLogEntries  []string
	}{
		{
			name: "nil-request.config",
			inBuildInstruction: buildInstruction{
				config: nil,
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedLogCalls:   []string{"logInfo"},
			expectedLogEntries: []string{"INFO: Rules engine tree manager shutting down"},
		},
		{
			name: "acount-id-found-but-rebuild-needed-because-entry-expired",
			inBuildInstruction: buildInstruction{
				config:    getValidJsonConfig(),
				accountID: "account-id-one",
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
					timestamp:    time.Now().Add(-1 * time.Hour),
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "801ee5db94088f300be22b6d3ab9bb546be66ebbc996ceadd534fdcc9992a958",
				},
			},
			expectedLogCalls:   []string{"logInfo"},
			expectedLogEntries: []string{"INFO: Rules engine tree manager shutting down"},
		},
		{
			name: "acount-id-found-rebuild-not-needed",
			inBuildInstruction: buildInstruction{
				config:    getValidJsonConfig(),
				accountID: "account-id-one",
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
					timestamp:    time.Now().Add(1 * time.Hour),
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedLogCalls:   []string{"logInfo"},
			expectedLogEntries: []string{"INFO: Rules engine tree manager shutting down"},
		},
		{
			name: "NewConfig-error",
			inBuildInstruction: buildInstruction{
				config:    getMalformedJsonConfig(),
				accountID: "account-id-two",
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedLogCalls: []string{"logInfo", "logError"},
			expectedLogEntries: []string{
				"INFO: Rules engine tree manager shutting down",
				"ERROR: Rules engine error parsing config for account account-id-two: JSON schema validation: invalid character 'm' looking for beginning of value",
			},
		},
		{
			name: "new-account-id-disabled-config",
			inBuildInstruction: buildInstruction{
				config:    getDisabledJsonConfig(),
				accountID: "account-id-two",
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedLogCalls: []string{"logInfo", "logInfo"},
			expectedLogEntries: []string{
				"INFO: Rules engine tree manager shutting down",
				"INFO: Rules engine disabled for account account-id-two",
			},
		},
		{
			name: "existing-account-id-needs-rebuild-but-new-config-is-disabled",
			inBuildInstruction: buildInstruction{
				config:    getDisabledJsonConfig(),
				accountID: "account-id-one",
			},
			inStoredDataInCache: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
			expectedCachedData: map[accountID]*cacheEntry{},
			expectedLogCalls:   []string{"logInfo", "logInfo"},
			expectedLogEntries: []string{
				"INFO: Rules engine tree manager shutting down",
				"INFO: Rules engine disabled for account account-id-one",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// set test
			var wg sync.WaitGroup
			wg.Add(len(tc.expectedLogCalls))
			logger := &mockLogger{
				wgDo: func() {
					wg.Done()
				},
			}
			for _, method := range tc.expectedLogCalls {
				logger.On(method, mock.Anything)
			}

			tm := &treeManager{
				done:            make(chan struct{}),
				requests:        make(chan buildInstruction, 10),
				schemaValidator: validator,
				monitor:         logger,
			}

			cache := NewCache()
			cache.m.Store(tc.inStoredDataInCache)

			// run
			go tm.Run(cache)
			time.Sleep(100 * time.Millisecond)
			tm.requests <- tc.inBuildInstruction
			tm.done <- struct{}{}
			wg.Wait()

			// assertions
			for _, method := range tc.expectedLogCalls {
				logger.AssertCalled(t, method)
			}

			assert.ElementsMatch(t, logger.logMsgs, tc.expectedLogEntries)

			actualCachedData := cache.m.Load().(map[accountID]*cacheEntry)
			if !assert.Len(t, actualCachedData, len(tc.expectedCachedData)) {
				return
			}
			for accountID, expectedEntry := range tc.expectedCachedData {
				actualEntry, exists := actualCachedData[accountID]
				if !assert.True(t, exists) {
					continue
				}
				assert.Equal(t, expectedEntry.hashedConfig, actualEntry.hashedConfig)
			}
		})
	}
}

type mockLogger struct {
	mock.Mock
	logMsgs []string
	wgDo    func()
}

func (logger *mockLogger) logError(msg string) {
	logger.logMsgs = append(logger.logMsgs, fmt.Sprintf("ERROR: %s", msg))
	logger.Called()
	logger.wgDo()
	return
}

func (logger *mockLogger) logInfo(msg string) {
	logger.logMsgs = append(logger.logMsgs, fmt.Sprintf("INFO: %s", msg))
	logger.Called()
	logger.wgDo()
	return
}

func getDisabledJsonConfig() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": false,
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
    "rulesets": [
      {
        "stage": "processed_auction_request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelgroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "deviceCountryIn",
                "args": {"countries": ["USA"]}
              },
              {
                "function": "dataCenterIn",
                "args": {"datacenters": ["us-east", "us-west"]}
              },
              {
                "function": "channel"
              }
            ],
            "default": [],
            "rules": [
              {
                "conditions": [
                  "true",
                  "true",
                  "amp"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderA"], "seatNonBid": 111}
                  }
                ]
              },
              {
                "conditions": [
                  "true",
                  "false",
                  "web"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": {"bidders": ["bidderB"], "seatNonBid": 222}
                  }
                ]
              },
              {
                "conditions": [
                  "false",
                  "false",
                  "*"
                ],
                "results": [
                  {
                    "function": "includeBidders",
                    "args": {"bidders": ["bidderC"], "seatNonBid": 333}
                  }
                ]
              }
            ]
          },
          {
            "weight": 1,
            "analyticsKey": "experiment-name",
            "version": "3.0",
            "schema": [{"function": "channel"}],
            "rules": [
              {
                "conditions": ["*"],
                "results": [{"function": "includeBidders", "args": {"bidders": ["bidderC"], "seatNonBid": 333}}]
              }
            ]
          }
        ]
      }
    ]
  }
`)
	return &rv
}

func getJsonConfigUnknownFunction() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": true,
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
    "rulesets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelgroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "unknownFunction",
                "args": ["USA"]
              },
              {
                "function": "dataCenters",
                "args": ["us-east", "us-west"]
              },
              {
                "function": "channel"
              }
            ],
            "default": [],
            "rules": [
              {
                "conditions": [
                  "true",
                  "true",
                  "amp"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": [{"bidders": ["bidderA"], "seatNonBid": 111}]
                  }
                ]
              },
              {
                "conditions": [
                  "true",
                  "false",
                  "web"
                ],
                "results": [
                  {
                    "function": "excludeBidders",
                    "args": [{"bidders": ["bidderB"], "seatNonBid": 222}]
                  }
                ]
              },
              {
                "conditions": [
                  "false",
                  "false",
                  "*"
                ],
                "results": [
                  {
                    "function": "includeBidders",
                    "args": [{"bidders": ["bidderC"], "seatNonBid": 333}]
                  }
                ]
              }
            ]
          },
          {
            "weight": 1,
            "analyticsKey": "experiment-name",
            "version": "3.0",
            "schema": [{"function": "channel"}],
            "rules": [
              {
                "conditions": ["*"],
                "results": [{"function": "includeBidders", "args": [{"bidders": ["bidderC"], "seatNonBid": 333}]}]
              }
            ]
          }
        ]
      }
    ]
  }
`)
	return &rv
}

func getMalformedJsonConfig() *json.RawMessage {
	rv := json.RawMessage(`malformed`)
	return &rv
}
