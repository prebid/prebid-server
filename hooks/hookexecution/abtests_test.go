package hookexecution

import (
	"sync"
	"testing"

	"github.com/prebid/prebid-server/v4/config"
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
