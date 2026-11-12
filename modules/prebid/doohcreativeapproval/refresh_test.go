package doohcreativeapproval

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalRefreshCoordinatorCoalescesAndLimitsLookups(t *testing.T) {
	coordinator := newApprovalRefreshCoordinator(1)
	first := approvalRefresh{Creative: creativeApproval{CreativeApprovalID: "v1:first"}}
	second := approvalRefresh{Creative: creativeApproval{CreativeApprovalID: "v1:second"}}

	claimed, capacityAvailable := coordinator.claim([]approvalRefresh{first})
	require.True(t, capacityAvailable)
	require.Equal(t, []approvalRefresh{first}, claimed)

	claimed, capacityAvailable = coordinator.claim([]approvalRefresh{first})
	assert.True(t, capacityAvailable)
	assert.Empty(t, claimed)

	claimed, capacityAvailable = coordinator.claim([]approvalRefresh{second})
	assert.False(t, capacityAvailable)
	assert.Empty(t, claimed)

	coordinator.finish([]approvalRefresh{first})

	claimed, capacityAvailable = coordinator.claim([]approvalRefresh{second})
	require.True(t, capacityAvailable)
	require.Equal(t, []approvalRefresh{second}, claimed)
	coordinator.finish([]approvalRefresh{second})
}

func TestApprovalRefreshCoordinatorSkipsCreativeAlreadyInFlight(t *testing.T) {
	coordinator := newApprovalRefreshCoordinator(2)
	refresh := approvalRefresh{Creative: creativeApproval{CreativeApprovalID: "v1:creative"}}

	claimed, capacityAvailable := coordinator.claim([]approvalRefresh{refresh})
	require.True(t, capacityAvailable)
	require.Len(t, claimed, 1)

	duplicateClaim, capacityAvailable := coordinator.claim([]approvalRefresh{refresh})
	assert.True(t, capacityAvailable)
	assert.Empty(t, duplicateClaim)
	assert.Equal(t, 1, len(coordinator.slots))

	coordinator.finish(claimed)
}
