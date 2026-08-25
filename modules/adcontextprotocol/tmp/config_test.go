package tmp

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

// genTestKey returns a fresh Ed25519 keypair in PKCS#8 PEM form, ready to drop
// into SigningConfig.PrivateKeyPEM. Kept here so every test can produce a valid
// key without pulling in adcp-go's helpers.
func genTestKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block))
}

func validConfig(t *testing.T) Config {
	return Config{
		SellerAgentURL: "https://seller.example.com",
		Signing: SigningConfig{
			KeyID:         "kid-1",
			PrivateKeyPEM: genTestKey(t),
		},
		PropertyRegistry: PropertyRegistryConfig{
			Endpoint:   "https://agenticadvertising.org/api/registry/resolve",
			Mode:       "resolve",
			AuthBearer: "test-bearer",
		},
		Providers: []ProviderConfig{
			{
				Name:        "example",
				IdentityURL: "https://tmp.example.com/identity",
				ContextURL:  "https://tmp.example.com/context",
			},
		},
	}
}

func TestValidated_Defaults(t *testing.T) {
	cfg := validConfig(t)
	if _, err := cfg.validated(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if cfg.TimeoutMs != 300 {
		t.Errorf("TimeoutMs default = %d, want 300", cfg.TimeoutMs)
	}
	if cfg.PropertyRegistry.CacheTTLSeconds != 3600 {
		t.Errorf("PropertyRegistry.CacheTTLSeconds default = %d, want 3600", cfg.PropertyRegistry.CacheTTLSeconds)
	}
	if cfg.TargetingKey != "adcp" {
		t.Errorf("TargetingKey default = %q, want %q", cfg.TargetingKey, "adcp")
	}
	if cfg.PackageTargetingKey != "adcp_package_id" {
		t.Errorf("PackageTargetingKey default = %q, want %q", cfg.PackageTargetingKey, "adcp_package_id")
	}
}

func TestValidated_PropertyRegistryDefaults(t *testing.T) {
	cfg := validConfig(t)
	cfg.PropertyRegistry.Endpoint = ""
	cfg.PropertyRegistry.Mode = ""
	cfg.PropertyRegistry.ProvenanceType = ""
	if _, err := cfg.validated(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if cfg.PropertyRegistry.Endpoint != "https://agenticadvertising.org/api/registry/resolve" {
		t.Errorf("endpoint default = %q", cfg.PropertyRegistry.Endpoint)
	}
	if cfg.PropertyRegistry.Mode != "resolve" {
		t.Errorf("mode default = %q, want resolve", cfg.PropertyRegistry.Mode)
	}
	if cfg.PropertyRegistry.ProvenanceType != "member_assertion" {
		t.Errorf("provenance_type default = %q, want member_assertion", cfg.PropertyRegistry.ProvenanceType)
	}
}

func TestValidated_PropertyRegistryRejectsBadMode(t *testing.T) {
	cfg := validConfig(t)
	cfg.PropertyRegistry.Mode = "contribute"
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error on invalid mode")
	}
}

func TestValidated_PropertyRegistryRejectsBadProvenance(t *testing.T) {
	cfg := validConfig(t)
	cfg.PropertyRegistry.ProvenanceType = "crawl" // reserved for server-side pipelines
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error on reserved provenance type")
	}
}

func TestValidated_PropertyRegistryResolveRequiresBearer(t *testing.T) {
	cfg := validConfig(t)
	cfg.PropertyRegistry.Mode = "resolve"
	cfg.PropertyRegistry.AuthBearer = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when mode=resolve without bearer")
	}
}

func TestValidated_PropertyRegistryLookupAllowsMissingBearer(t *testing.T) {
	cfg := validConfig(t)
	cfg.PropertyRegistry.Mode = "lookup"
	cfg.PropertyRegistry.AuthBearer = ""
	if _, err := cfg.validated(); err != nil {
		t.Fatalf("expected mode=lookup without bearer to be valid; got %v", err)
	}
}

func TestValidated_ProviderRejectsMismatchedBaseURLs(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = "https://a.example.com/identity"
	cfg.Providers[0].ContextURL = "https://b.example.com/context"
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when identity_url and context_url derive to different base URLs")
	}
}

func TestValidated_ProviderAcceptsSameBaseURLs(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = "https://tmp.example.com/identity"
	cfg.Providers[0].ContextURL = "https://tmp.example.com/context"
	if _, err := cfg.validated(); err != nil {
		t.Fatalf("expected valid config; got %v", err)
	}
}

func TestProviderSigningBase(t *testing.T) {
	// signingBase's callers must only ever see URLs validated() has
	// already accepted — i.e. URLs whose path ends in /identity or
	// /context after trailing-slash normalization. This table covers
	// what the helper does with those valid shapes.
	cases := []struct {
		name        string
		identityURL string
		contextURL  string
		want        string
	}{
		{"identity-only", "https://tmp.example.com/identity", "", "https://tmp.example.com"},
		{"context-only", "", "https://tmp.example.com/context", "https://tmp.example.com"},
		{"both-match", "https://tmp.example.com/identity", "https://tmp.example.com/context", "https://tmp.example.com"},
		{"trailing-slash-normalized", "https://tmp.example.com/identity/", "", "https://tmp.example.com"},
		{"nested-path-preserved", "https://tmp.example.com/v1/identity", "", "https://tmp.example.com/v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := ProviderConfig{
				IdentityURL: tc.identityURL,
				ContextURL:  tc.contextURL,
			}
			if got := p.signingBase(); got != tc.want {
				t.Errorf("signingBase() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestValidated_ProviderRejectsIdentityURLWithoutSuffix(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = "https://tmp.example.com/api/identity-match"
	cfg.Providers[0].ContextURL = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when identity_url does not end in /identity")
	}
}

func TestValidated_ProviderRejectsContextURLWithoutSuffix(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = ""
	cfg.Providers[0].ContextURL = "https://tmp.example.com/api/context-match"
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when context_url does not end in /context")
	}
}

func TestValidated_ProviderRejectsCaseWrongSuffix(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = "https://tmp.example.com/IDENTITY"
	cfg.Providers[0].ContextURL = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when identity_url suffix is not literal /identity (case-sensitive)")
	}
}

func TestValidated_ProviderNeedsAtLeastOneURL(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].IdentityURL = ""
	cfg.Providers[0].ContextURL = ""
	_, err := cfg.validated()
	if err == nil {
		t.Fatal("expected error when both provider URLs are empty")
	}
	if !strings.Contains(err.Error(), "at least one of identity_url or context_url") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidated_MissingSellerAgentURL(t *testing.T) {
	cfg := validConfig(t)
	cfg.SellerAgentURL = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error for missing seller_agent_url")
	}
}

func TestValidated_MissingSigningKey(t *testing.T) {
	cfg := validConfig(t)
	cfg.Signing.PrivateKeyPEM = ""
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error for missing signing.private_key_pem")
	}
}

func TestValidated_SigningDisabledAllowsEmptyKey(t *testing.T) {
	cfg := validConfig(t)
	cfg.Signing.KeyID = ""
	cfg.Signing.PrivateKeyPEM = ""
	cfg.Signing.Disabled = true
	priv, err := cfg.validated()
	if err != nil {
		t.Fatalf("expected valid config with signing disabled, got %v", err)
	}
	if priv != nil {
		t.Errorf("expected nil private key when signing disabled; got %v", priv)
	}
}

func TestValidated_SigningDisabledRejectsStaleKeyID(t *testing.T) {
	cfg := validConfig(t)
	cfg.Signing.Disabled = true
	cfg.Signing.PrivateKeyPEM = ""
	// KeyID left set from validConfig — stale material next to disabled=true.
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when signing.disabled=true but key_id is still populated")
	}
}

func TestValidated_SigningDisabledRejectsStalePEM(t *testing.T) {
	cfg := validConfig(t)
	cfg.Signing.Disabled = true
	cfg.Signing.KeyID = ""
	// PrivateKeyPEM left set from validConfig — stale material next to disabled=true.
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when signing.disabled=true but private_key_pem is still populated")
	}
}

func TestValidated_LatLongPrecisionCapped(t *testing.T) {
	cfg := validConfig(t)
	cfg.Masking.Enabled = true
	cfg.Masking.Geo.LatLongPrecision = 5
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when lat_long_precision > 4")
	}
}

func TestValidated_MaskingDefaultEIDList(t *testing.T) {
	cfg := validConfig(t)
	cfg.Masking.Enabled = true
	if _, err := cfg.validated(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if len(cfg.Masking.User.PreserveEids) == 0 {
		t.Fatal("expected default EID list to be populated when masking is enabled")
	}
}

func TestValidated_TmpxMacroMappingRejectsUnknownProvider(t *testing.T) {
	cfg := validConfig(t)
	cfg.TmpxMacroMapping = map[string]map[string]string{
		"other_provider": {"primary": "TMPX_1"},
	}
	_, err := cfg.validated()
	if err == nil {
		t.Fatal("expected error when tmpx_macro_mapping references an unknown provider")
	}
	if !strings.Contains(err.Error(), "other_provider") {
		t.Errorf("expected error to name the offending provider; got %v", err)
	}
}

func TestValidated_ProviderNameRejectsSpecCharsetViolation(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].Name = "has-a-hyphen"
	_, err := cfg.validated()
	if err == nil {
		t.Fatal("expected error when provider name uses a char outside the adcp provider_id charset")
	}
}

func TestValidated_TmpxSlotsOverV1Cap(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].TmpxSlots = []string{"a", "b", "c"}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when tmpx_slots exceeds v1 cap")
	}
}

func TestValidated_TmpxSlotIDCharsetEnforced(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].TmpxSlots = []string{"1_leading_digit"}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when a slot_id fails the adcp charset")
	}
}

func TestValidated_TmpxSlotsRejectDuplicates(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].TmpxSlots = []string{"primary", "primary"}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when tmpx_slots has duplicates")
	}
}

func TestValidated_TmpxMacroMappingRejectsSlotNotInRegistration(t *testing.T) {
	cfg := validConfig(t)
	cfg.Providers[0].TmpxSlots = []string{"primary"}
	cfg.TmpxMacroMapping = map[string]map[string]string{
		"example": {"unregistered": "TMPX_X"},
	}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when mapping references a slot_id the provider did not register")
	}
}

func TestValidated_TmpxMacroMappingSlotIDCharsetEnforced(t *testing.T) {
	cfg := validConfig(t)
	cfg.TmpxMacroMapping = map[string]map[string]string{
		"example": {"1_bad_slot": "TMPX_1"},
	}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when mapping slot_id fails the adcp charset")
	}
}

func TestValidated_TmpxMacroMappingRejectsEmptyMacro(t *testing.T) {
	cfg := validConfig(t)
	cfg.TmpxMacroMapping = map[string]map[string]string{
		"example": {"primary": ""},
	}
	if _, err := cfg.validated(); err == nil {
		t.Fatal("expected error when destination macro is empty")
	}
}

func TestValidated_TmpxMacroMappingAcceptsValidEntry(t *testing.T) {
	cfg := validConfig(t)
	cfg.TmpxMacroMapping = map[string]map[string]string{
		"example": {"primary": "TMPX_1", "secondary": "TMPX_2"},
	}
	if _, err := cfg.validated(); err != nil {
		t.Errorf("expected valid config, got %v", err)
	}
}
