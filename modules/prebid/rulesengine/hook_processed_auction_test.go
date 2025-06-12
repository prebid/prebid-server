package rulesengine

import (
	"fmt"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleProcessedAuctionHook(t *testing.T) {
	tests := []struct {
		name           string
		ruleSets       []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]
		payload        hs.ProcessedAuctionRequestPayload
		expectedError  bool
		expectedResult hs.HookResult[hs.ProcessedAuctionRequestPayload]
	}{
		{
			name:           "empty-rule-sets",
			ruleSets:       []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
			payload:        hs.ProcessedAuctionRequestPayload{},
			expectedError:  false,
			expectedResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{},
		},
		{
			name: "single-rule-set-with-valid-model-group-and-empty-rule-tree",
			ruleSets: []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				{
					modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
						{
							weight: 100,
							tree:   rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
						},
					},
				},
			},
			payload:       hs.ProcessedAuctionRequestPayload{},
			expectedError: false,
			expectedResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				Errors: []string{"tree root is nil"},
			},
		},
		{
			name: "failed-to-select-model-group",
			ruleSets: []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				{
					modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
				},
			},
			payload:       hs.ProcessedAuctionRequestPayload{},
			expectedError: false,
			expectedResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				Errors: []string{"failed to select model group: no model groups available"},
			},
		},
		{
			name: "single-rule-set-with-valid-model-group-and-valid-rule-tree",
			ruleSets: []cacheRuleSet[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
				{
					modelGroups: []cacheModelGroup[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
						{
							weight: 100,
							tree: rules.Tree[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{
								Root: &rules.Node[openrtb_ext.RequestWrapper, hs.HookResult[hs.ProcessedAuctionRequestPayload]]{},
							},
						},
					},
				},
			},
			payload:        hs.ProcessedAuctionRequestPayload{},
			expectedError:  false,
			expectedResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleProcessedAuctionHook(tt.ruleSets, tt.payload)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestSelectModelGroup(t *testing.T) {
	tests := []struct {
		name          string
		modelGroups   []ModelGroup
		mockRandValue int
		expectedIndex int
		expectedError error
	}{
		{
			name:          "empty-model-groups",
			modelGroups:   []ModelGroup{},
			expectedError: fmt.Errorf("no model groups available"),
		},
		{
			name: "single-model-group",
			modelGroups: []ModelGroup{
				{weight: 100, analyticsKey: "group1"},
			},
			expectedIndex: 0,
			expectedError: nil,
		},
		{
			name: "equal-weights-first-selected",
			modelGroups: []ModelGroup{
				{weight: 50, analyticsKey: "group1"},
				{weight: 50, analyticsKey: "group2"},
			},
			mockRandValue: 25,
			expectedIndex: 0,
			expectedError: nil,
		},
		{
			name: "equal-weights-second-selected",
			modelGroups: []ModelGroup{
				{weight: 50, analyticsKey: "group1"},
				{weight: 50, analyticsKey: "group2"},
			},
			mockRandValue: 75,
			expectedIndex: 1,
			expectedError: nil,
		},
		{
			name: "uneven-weights-first-selected",
			modelGroups: []ModelGroup{
				{weight: 70, analyticsKey: "group1"},
				{weight: 30, analyticsKey: "group2"},
			},
			mockRandValue: 65,
			expectedIndex: 0,
			expectedError: nil,
		},
		{
			name: "uneven-weights-second-selected",
			modelGroups: []ModelGroup{
				{weight: 70, analyticsKey: "group1"},
				{weight: 30, analyticsKey: "group2"},
			},
			mockRandValue: 75,
			expectedIndex: 1,
			expectedError: nil,
		},
		{
			name: "three-groups-with-middle-selected",
			modelGroups: []ModelGroup{
				{weight: 20, analyticsKey: "group1"},
				{weight: 30, analyticsKey: "group2"},
				{weight: 50, analyticsKey: "group3"},
			},
			mockRandValue: 35,
			expectedIndex: 1,
			expectedError: nil,
		},
		{
			name: "zero-weight-defaults-to-100-first-selected",
			modelGroups: []ModelGroup{
				{weight: 0, analyticsKey: "group1"},
				{weight: 50, analyticsKey: "group2"},
			},
			mockRandValue: 80, // Will select second group (101-150)
			expectedIndex: 0,
			expectedError: nil,
		},
		{
			name: "zero-weight-defaults-to-100-second-selected",
			modelGroups: []ModelGroup{
				{weight: 0, analyticsKey: "group1"},
				{weight: 50, analyticsKey: "group2"},
			},
			mockRandValue: 120,
			expectedIndex: 1,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRandom := &mockRandomGenerator{returnValue: tt.mockRandValue}

			result, err := selectModelGroup(tt.modelGroups, mockRandom)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.modelGroups[tt.expectedIndex].analyticsKey, result.analyticsKey)
			}
		})
	}
}

// mockRandomGenerator implements randomutil.RandomGenerator for testing
type mockRandomGenerator struct {
	returnValue int
}

func (g *mockRandomGenerator) Intn(n int) int {
	// Ensure return value is within the range [0,n)
	if g.returnValue >= n {
		return g.returnValue % n
	}
	return g.returnValue
}

func (g *mockRandomGenerator) GenerateInt63() int64 {
	return int64(g.returnValue)
}
