package exchange

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var testAdapter = BidderAdapter{
	me: &config.NilMetricsEngine{},
	config: bidderAdapterConfig{
		ThrottleConfig: bidderAdapterThrottleConfig{
			enabled:        true,
			throttleWindow: 100,
			bulkValue:      0.99,
			deltaValue:     0.01,
		},
	},
}

func TestBidderAdapter_LogHealthCheck(t *testing.T) {
	testCases := []struct {
		name          string
		initialHealth float64
		success       bool
		expectedRange struct {
			min float64
			max float64
		}
	}{
		{
			name:          "success with zero initial health",
			initialHealth: 0.0,
			success:       true,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: -0.001, // Allow for small floating point errors
				max: 0.001,
			},
		},
		{
			name:          "failure with zero initial health",
			initialHealth: 0.0,
			success:       false,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: 0.009, // 0.99*0 + 0.01 = 0.01
				max: 0.011, // Allow for small floating point errors
			},
		},
		{
			name:          "success with 0.5 initial health",
			initialHealth: 0.5,
			success:       true,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: 0.485, // 0.99*0.5 = 0.495
				max: 0.505, // Allow for small floating point errors
			},
		},
		{
			name:          "failure with 0.5 initial health",
			initialHealth: 0.5,
			success:       false,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: 0.495, // 0.99*0.5 + 0.01 = 0.505
				max: 0.515, // Allow for small floating point errors
			},
		},
		{
			name:          "success with 1.0 initial health",
			initialHealth: 1.0,
			success:       true,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: 0.98, // 0.99*1.0 = 0.99
				max: 1.0,  // Allow for small floating point errors
			},
		},
		{
			name:          "failure with 1.0 initial health",
			initialHealth: 1.0,
			success:       false,
			expectedRange: struct {
				min float64
				max float64
			}{
				min: 0.99,  // 0.99*1.0 + 0.01 = 1.0
				max: 1.001, // Allow for small floating point errors
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new BidderAdapter with the initial health value
			bidder := &BidderAdapter{}
			*bidder = testAdapter
			bidder.healthBits = math.Float64bits(tc.initialHealth)

			// Call the logHealthCheck method
			bidder.logHealthCheck(tc.success)

			// Check the result is within expected range
			actual := bidder.getHealth()
			assert.True(t, actual >= tc.expectedRange.min && actual <= tc.expectedRange.max,
				"Health value %f should be between %f and %f", actual, tc.expectedRange.min, tc.expectedRange.max)
		})
	}
}

func TestBidderAdapter_LogHealthCheck_ConcurrentAccess(t *testing.T) {
	bidder := &BidderAdapter{}
	*bidder = testAdapter
	bidder.healthBits = math.Float64bits(0.5) // Start at 0.5

	const numGoroutines = 100
	const updatesPerGoroutine = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines to update health concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(routineNum int) {
			defer wg.Done()
			// Alternate between success and failure
			success := routineNum%2 == 0
			for j := 0; j < updatesPerGoroutine; j++ {
				bidder.logHealthCheck(success)
			}
		}(i)
	}

	wg.Wait()

	// Check that the final health value is valid
	health := bidder.getHealth()
	assert.False(t, math.IsNaN(health), "Health should not be NaN")
	assert.False(t, math.IsInf(health, 0), "Health should not be infinite")
	assert.True(t, health >= 0, "Health should not be negative")
}

func TestBidderAdapter_ShouldRequest(t *testing.T) {
	testCases := []struct {
		name           string
		healthValue    float64
		expectedAlways bool
		description    string
	}{
		{
			name:           "health below threshold",
			healthValue:    0.1,
			expectedAlways: true,
			description:    "When health < 0.2, shouldRequest should always return true",
		},
		{
			name:           "health at threshold",
			healthValue:    0.21,
			expectedAlways: false,
			description:    "When health = 0.2, shouldRequest should be probabilistic",
		},
		{
			name:           "health above threshold",
			healthValue:    0.6,
			expectedAlways: false,
			description:    "When health > 0.2, shouldRequest should be probabilistic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new BidderAdapter with the given health value
			meMock := metrics.MetricsEngineMock{}
			var biddername openrtb_ext.BidderName
			meMock.On("RecordAdapterThrottled", biddername).Return()
			bidder := &BidderAdapter{}
			*bidder = testAdapter
			bidder.me = &meMock
			atomic.StoreUint64(&bidder.healthBits, math.Float64bits(tc.healthValue))

			if tc.expectedAlways {
				// Test multiple times to ensure it's consistently returning true
				for i := 0; i < 100; i++ {
					result := bidder.shouldRequest()
					assert.True(t, result, tc.description)
				}
			} else {
				// For probabilistic cases, we'll test that it sometimes returns false by running many iterations
				const numTests = 1000
				throttledCount := 0
				passedCount := 0

				for i := 0; i < numTests; i++ {
					if bidder.shouldRequest() {
						passedCount++
						// t.Log("Health check returned true on iteration", i, "for health value", tc.healthValue)
					} else {
						throttledCount++
						// t.Log("Health check returned false on iteration", i, "for health value", tc.healthValue)
					}
				}

				// We should see at least some false responses if the health is above 0.2
				if tc.healthValue > 0.2 {
					assert.True(t, throttledCount > 0, "Expected at least some false responses for health %f", tc.healthValue)
					meMock.AssertNumberOfCalls(t, "RecordAdapterThrottled", throttledCount)
				}
			}
		})
	}
}

func TestBidderAdapter_HealthIntegration(t *testing.T) {
	// Test integration between logHealthCheck and shouldRequest
	bidder := &BidderAdapter{}
	*bidder = testAdapter

	// Start with zero health - should always request
	bidder.healthBits = math.Float64bits(0.0)

	// Check initial state
	assert.Equal(t, float64(0.0), bidder.getHealth())
	assert.True(t, bidder.shouldRequest(), "Should initially always request with zero health")

	// Log many failures to increase health
	for i := 0; i < 100; i++ {
		bidder.logHealthCheck(false)
	}

	// Now health should be higher, near 1.0
	health := bidder.getHealth()
	assert.True(t, health > 0.5, "Health should increase with failures (current: %f)", health)

	// With higher health, some requests should be throttled
	// Since shouldRequest has randomness, we need to run it many times
	const numTests = 1000
	throttledCount := 0
	for i := 0; i < numTests; i++ {
		rand.Seed(int64(i)) // For deterministic testing
		if !bidder.shouldRequest() {
			throttledCount++
		}
	}

	// We should see some throttled requests now
	assert.True(t, throttledCount > 0, "Expected some requests to be throttled with high health")
	t.Logf("Health: %f, Throttled: %d/%d", health, throttledCount, numTests)

	// Now log many successes to decrease health
	for i := 0; i < 100; i++ {
		bidder.logHealthCheck(true)
	}

	// Health should decrease
	newHealth := bidder.getHealth()
	assert.True(t, newHealth < health, "Health should decrease with successes (before: %f, after: %f)", health, newHealth)
}

func TestBidderAdapter_GetHealth(t *testing.T) {
	testValues := []float64{0.0, 0.1, 0.5, 0.99, 1.0}

	for _, val := range testValues {
		bidder := &BidderAdapter{}
		*bidder = testAdapter
		bidder.healthBits = math.Float64bits(val)

		actual := bidder.getHealth()
		assert.InDelta(t, val, actual, 0.000001, "getHealth() should return the stored health value")
	}
}
