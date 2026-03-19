package hookexecution

import (
	"sync"
	"testing"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/stretchr/testify/assert"
)

func TestRaceABTests(t *testing.T) {
	const numGoroutines = 20
	const module = "vendor.module"

	enabled := true

	cfg := &config.Configuration{
		Hooks: config.Hooks{
			HostExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{
					{
						ModuleCode:        module,
						Enabled:           &enabled,
						AdServerTargeting: "target",
					},
				},
			},
		},
	}

	ab := NewABTests(cfg)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ab.Run(module)
			outcome := &StageOutcome{}
			ab.WriteOutcome(outcome)
		}()
	}

	wg.Wait()

	_ = ab.GetTargetingKeywords()
}

func TestABTestsCheckAndSetLoggedNoDuplicates(t *testing.T) {
	const numGoroutines = 20
	const module = "vendor.module"

	enabled := true
	pa := uint16(0) // always skip so we can count fallback group additions

	cfg := &config.Configuration{
		Hooks: config.Hooks{
			HostExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{
					{
						ModuleCode:    module,
						Enabled:       &enabled,
						PercentActive: &pa,
					},
				},
			},
		},
	}

	ab := NewABTests(cfg)

	// Shared outcome — all goroutines append to the same outcome.
	// The checkAndSetLogged guard ensures only one goroutine appends.
	outcome := &StageOutcome{}

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			localOutcome := &StageOutcome{}
			ab.WriteOutcome(localOutcome)

			mu.Lock()
			outcome.Groups = append(outcome.Groups, localOutcome.Groups...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// With checkAndSetLogged, only one goroutine should have added the fallback group.
	count := 0
	for _, g := range outcome.Groups {
		for _, ir := range g.InvocationResults {
			if ir.HookID.ModuleCode == module {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 analytics entry for module %q, got %d", module, count)
	}
}

func TestPlanAccountDisabledClearsLogMap(t *testing.T) {
	const module = "vendor.module"

	hostEnabled := true
	accountEnabled := false

	cfg := &config.Configuration{
		Hooks: config.Hooks{
			HostExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{
					{
						ModuleCode: module,
						Enabled:    &hostEnabled,
					},
				},
			},
			DefaultAccountExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{
					{
						ModuleCode: module,
						Enabled:    &accountEnabled,
					},
				},
			},
		},
	}

	ab := NewABTests(cfg)
	outcome := &StageOutcome{}
	ab.WriteOutcome(outcome)

	// Account plan disabled the module: logMap must be cleared.
	// WriteOutcome must not emit any analytics entry for the module.
	for _, g := range outcome.Groups {
		for _, ir := range g.InvocationResults {
			if ir.HookID.ModuleCode == module {
				t.Errorf("expected no analytics entry for disabled module %q, but got one", module)
			}
		}
	}
}

func TestSetAccountID(t *testing.T) {
	tests := []struct {
		description string
		body        []byte
		expected    string
	}{
		{
			description: "site.publisher.id is used when present",
			body:        []byte(`{"site":{"publisher":{"id":"site-pub-id"}}}`),
			expected:    "site-pub-id",
		},
		{
			description: "app.publisher.id is used as fallback when site.publisher.id is absent",
			body:        []byte(`{"app":{"publisher":{"id":"app-pub-id"}}}`),
			expected:    "app-pub-id",
		},
		{
			description: "site.publisher.id takes precedence over app.publisher.id",
			body:        []byte(`{"site":{"publisher":{"id":"site-pub-id"}},"app":{"publisher":{"id":"app-pub-id"}}}`),
			expected:    "site-pub-id",
		},
		{
			description: "empty result when neither site nor app publisher id present",
			body:        []byte(`{"imp":[{"id":"imp-1"}]}`),
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ab := NewABTests(&config.Configuration{})
			ab.SetAccountID(tc.body)
			assert.Equal(t, tc.expected, ab.accountID)
		})
	}
}
