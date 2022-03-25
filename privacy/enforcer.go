package privacy

// PolicyEnforcer determines if personally identifiable information (PII) should be removed or anonymized per the policy.
type PolicyEnforcer interface {
	// CanEnforce returns true when policy information is specifically provided by the publisher.
	CanEnforce() bool

	// ShouldEnforce returns true when the OpenRTB request should have personally identifiable
	// information (PII) removed or anonymized per the policy.
	ShouldEnforce(bidder string) bool
}

// NilPolicyEnforcer implements the PolicyEnforcer interface but will always return false.
type NilPolicyEnforcer struct{}

// CanEnforce is hardcoded to always return false.
func (NilPolicyEnforcer) CanEnforce() bool {
	return false
}

// ShouldEnforce is hardcoded to always return false.
func (NilPolicyEnforcer) ShouldEnforce(bidder string) bool {
	return false
}

// EnabledPolicyEnforcer decorates a PolicyEnforcer with an enabled flag.
type EnabledPolicyEnforcer struct {
	Enabled        bool
	PolicyEnforcer PolicyEnforcer
}

// CanEnforce returns true when the PolicyEnforcer can enforce.
func (p EnabledPolicyEnforcer) CanEnforce() bool {
	return p.PolicyEnforcer.CanEnforce()
}

// ShouldEnforce returns true when the enforcer is enabled the PolicyEnforcer allows enforcement.
func (p EnabledPolicyEnforcer) ShouldEnforce(bidder string) bool {
	if p.Enabled {
		return p.PolicyEnforcer.ShouldEnforce(bidder)
	}
	return false
}
