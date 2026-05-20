package hookexecution

import (
	"sync"
	"testing"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
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
			description: "dooh.publisher.id is used as fallback when site and app publisher id are absent",
			body:        []byte(`{"dooh":{"publisher":{"id":"dooh-pub-id"}}}`),
			expected:    "dooh-pub-id",
		},
		{
			description: "empty result when neither site nor app nor dooh publisher id present",
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

func TestPercentActiveBoundaries(t *testing.T) {
	orig := randIntn
	defer func() { randIntn = orig }()

	const module = "vendor.module"
	enabled := true

	tests := []struct {
		description string
		pa          uint16
		randVal     int
		wantRun     bool
	}{
		{description: "pa=50 rand=49 runs (49 < 50)", pa: 50, randVal: 49, wantRun: true},
		{description: "pa=50 rand=50 skips (50 < 50 false)", pa: 50, randVal: 50, wantRun: false},
		{description: "pa=1 rand=0 runs (0 < 1)", pa: 1, randVal: 0, wantRun: true},
		{description: "pa=1 rand=1 skips (1 < 1 false)", pa: 1, randVal: 1, wantRun: false},
		{description: "pa=99 rand=98 runs (98 < 99)", pa: 99, randVal: 98, wantRun: true},
		{description: "pa=99 rand=99 skips (99 < 99 false)", pa: 99, randVal: 99, wantRun: false},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			pa := tc.pa
			randIntn = func(_ int) int { return tc.randVal }

			cfg := &config.Configuration{
				Hooks: config.Hooks{
					HostExecutionPlan: config.HookExecutionPlan{
						ABTests: []config.ABTest{{
							ModuleCode:    module,
							Enabled:       &enabled,
							PercentActive: &pa,
						}},
					},
				},
			}
			ab := NewABTests(cfg)
			assert.Equal(t, tc.wantRun, ab.Run(module))
		})
	}
}

func TestWriteOutcomeNoDuplicatesForRunModule(t *testing.T) {
	const module = "vendor.module"
	enabled := true
	lat := true

	cfg := &config.Configuration{
		Hooks: config.Hooks{
			HostExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{{
					ModuleCode:      module,
					Enabled:         &enabled,
					LogAnalyticsTag: &lat,
					// PercentActive nil → defaults to 100 → always run
				}},
			},
		},
	}

	tests := []struct {
		description    string
		stageCount     int
		moduleInGroups bool // whether each stage outcome pre-populates an invocation result for the module
	}{
		{
			description:    "run module in groups across 1 stage produces 1 tag",
			stageCount:     1,
			moduleInGroups: true,
		},
		{
			description:    "run module in groups across 3 stages produces exactly 1 tag total",
			stageCount:     3,
			moduleInGroups: true,
		},
		{
			description:    "run module not in groups across 3 stages produces 0 tags (no synthetic entry for run modules)",
			stageCount:     3,
			moduleInGroups: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ab := NewABTests(cfg)

			totalTagCount := 0
			for range tc.stageCount {
				outcome := &StageOutcome{}
				if tc.moduleInGroups {
					outcome.Groups = []GroupOutcome{{
						InvocationResults: []HookOutcome{
							{HookID: HookID{ModuleCode: module}},
						},
					}}
				}
				ab.WriteOutcome(outcome)
				for _, g := range outcome.Groups {
					for _, ir := range g.InvocationResults {
						if ir.HookID.ModuleCode == module {
							totalTagCount += len(ir.AnalyticsTags.Activities)
						}
					}
				}
			}

			wantCount := 0
			if tc.moduleInGroups {
				wantCount = 1
			}
			assert.Equal(t, wantCount, totalTagCount,
				"analytics tag count across %d stage(s)", tc.stageCount)
		})
	}
}

func TestWriteOutcomeActivityContent(t *testing.T) {
	const module = "vendor.module"
	enabled := true
	lat := true

	cfg := &config.Configuration{
		Hooks: config.Hooks{
			HostExecutionPlan: config.HookExecutionPlan{
				ABTests: []config.ABTest{{
					ModuleCode:      module,
					Enabled:         &enabled,
					LogAnalyticsTag: &lat,
				}},
			},
		},
	}

	tests := []struct {
		description    string
		moduleInGroups bool
		wantStatus     hookanalytics.ResultStatus
		wantGroups     int
	}{
		{
			description:    "run module in groups gets ResultStatusRun tag",
			moduleInGroups: true,
			wantStatus:     hookanalytics.ResultStatusRun,
			wantGroups:     1,
		},
		{
			description:    "skipped module (pa=0) not in groups gets ResultStatusSkip synthetic entry",
			moduleInGroups: false,
			wantStatus:     hookanalytics.ResultStatusSkip,
			wantGroups:     1, // synthetic fallback group added
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			ab := NewABTests(cfg)

			if tc.wantStatus == hookanalytics.ResultStatusSkip {
				// Override to always-skip for this sub-test.
				pa := uint16(0)
				ab.config.Hooks.HostExecutionPlan.ABTests[0].PercentActive = &pa
				// Reset initOnce so the new pa takes effect.
				ab = NewABTests(&config.Configuration{
					Hooks: config.Hooks{
						HostExecutionPlan: config.HookExecutionPlan{
							ABTests: []config.ABTest{{
								ModuleCode:      module,
								Enabled:         &enabled,
								LogAnalyticsTag: &lat,
								PercentActive:   &pa,
							}},
						},
					},
				})
			}

			outcome := &StageOutcome{}
			if tc.moduleInGroups {
				outcome.Groups = []GroupOutcome{{
					InvocationResults: []HookOutcome{
						{HookID: HookID{ModuleCode: module}},
					},
				}}
			}
			ab.WriteOutcome(outcome)

			assert.Len(t, outcome.Groups, tc.wantGroups)

			var activity *hookanalytics.Activity
			for _, g := range outcome.Groups {
				for _, ir := range g.InvocationResults {
					if ir.HookID.ModuleCode == module {
						if len(ir.AnalyticsTags.Activities) > 0 {
							a := ir.AnalyticsTags.Activities[0]
							activity = &a
						}
					}
				}
			}

			if assert.NotNil(t, activity, "expected analytics activity for module %q", module) {
				assert.Equal(t, "core-module-abtests", activity.Name)
				assert.Equal(t, hookanalytics.ActivityStatusSuccess, activity.Status)
				if assert.Len(t, activity.Results, 1) {
					assert.Equal(t, tc.wantStatus, activity.Results[0].Status)
					assert.Equal(t, module, activity.Results[0].Values["module"])
				}
			}
		})
	}
}
