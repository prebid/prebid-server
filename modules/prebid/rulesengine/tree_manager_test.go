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
	schemaFile := "config/rules-engine-schema.json"
	validator, err := config.CreateSchemaValidator(schemaFile)
	assert.NoError(t, err, fmt.Sprintf("could not create schema validator using file %s", schemaFile))

	testCases := []struct {
		desc                string
		inBuildInstruction  buildInstruction
		inStoredDataInCache map[accountID]*cacheEntry
		expectedCachedData  map[accountID]*cacheEntry
		expectedLogEntries  []string
	}{
		{
			desc: "nil request.config",
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
			expectedLogEntries: []string{"logInfo"},
		},
		{
			desc: "acount-id found but rebuild needed because entry expired",
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
					hashedConfig: "05c533fada1f181b69b7b1d130e5c614c1da389bbdfeae99bf65e7940803b8ac",
				},
			},
			expectedLogEntries: []string{"logInfo"},
		},
		{
			desc: "acount-id found rebuild not needed",
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
			expectedLogEntries: []string{"logInfo"},
		},
		{
			desc: "NewConfig error",
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
			expectedLogEntries: []string{"logInfo", "logError"},
		},
		{
			desc: "new account-id disabled config",
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
			expectedLogEntries: []string{"logInfo", "logInfo"},
		},
		{
			desc: "existing account id needs rebuild but new config is disabled",
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
			expectedLogEntries: []string{"logInfo", "logInfo"},
		},
		{
			desc: "NewCacheEntry error",
			inBuildInstruction: buildInstruction{
				config:    getJsonConfigUnknownFunction(),
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
			expectedLogEntries: []string{"logInfo", "logError"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// set test
			var wg sync.WaitGroup
			wg.Add(len(tc.expectedLogEntries))
			logger := &mockLogger{
				f: func() {
					wg.Done()
				},
			}
			for _, method := range tc.expectedLogEntries {
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
			for _, method := range tc.expectedLogEntries {
				logger.AssertCalled(t, method)
			}

			actualCachedData := cache.m.Load().(map[accountID]*cacheEntry)
			//if assert.Len(t, actualCachedData, len(tc.expectedCachedData)) {
			if !assert.Len(t, actualCachedData, len(tc.expectedCachedData)) {
				return
			}
			for accountID, expectedEntry := range tc.expectedCachedData {
				//for accountID, _ := range tc.expectedCachedData {
				actualEntry, exists := actualCachedData[accountID]
				//_, exists := actualCachedData[accountID]
				if !assert.True(t, exists) {
					continue
				}
				assert.Equal(t, expectedEntry.hashedConfig, actualEntry.hashedConfig)
				assert.ElementsMatch(t, expectedEntry.ruleSetsForProcessedAuctionRequestStage, actualEntry.ruleSetsForProcessedAuctionRequestStage)
				//if !assert.Len(t, expectedEntry.ruleSetsForProcessedAuctionRequestStage, len(actualEntry.ruleSetsForProcessedAuctionRequestStage)) {
				//	continue
				//}
				//for i, expectedRuleSet := range expectedEntry.ruleSetsForProcessedAuctionRequestStage {
				//	assert.Equal(t, expectedRuleSet, actualEntry.ruleSetsForProcessedAuctionRequestStage[i])
				//}
			}
		})
	}
}

type mockLogger struct {
	mock.Mock
	f func()
}

func (logger *mockLogger) logError(msg string) {
	logger.Called()
	logger.f()
	return
}

func (logger *mockLogger) logInfo(msg string) {
	logger.Called()
	logger.f()
	return
}

func getDisabledJsonConfig() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": false,
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
    "ruleSets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelGroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "deviceCountry",
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

func getValidJsonConfig() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": true,
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
    "ruleSets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelGroups": [
          {
            "weight": 100,
            "analyticsKey": "experiment-name",
            "version": "4567",
            "schema": [
              {
                "function": "deviceCountry",
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

func getJsonConfigUnknownFunction() *json.RawMessage {
	rv := json.RawMessage(`
  {
    "enabled": true,
    "generateRulesFromBidderConfig": true,
    "timestamp": "20250131 00:00:00",
    "ruleSets": [
      {
        "stage": "processed-auction-request",
        "name": "exclude-in-jpn",
        "version": "1234",
        "modelGroups": [
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
