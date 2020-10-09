package privacy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilEnforcerCanEnforce(t *testing.T) {
	nilEnforcer := &NilPolicyEnforcer{}
	assert.False(t, nilEnforcer.CanEnforce())
}

func TestNilEnforcerShouldEnforce(t *testing.T) {
	nilEnforcer := &NilPolicyEnforcer{}
	assert.False(t, nilEnforcer.ShouldEnforce(""))
	assert.False(t, nilEnforcer.ShouldEnforce("anyBidder"))
}
