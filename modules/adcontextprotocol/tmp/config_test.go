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
			Endpoint: "https://agenticadvertising.org/api/properties/resolve",
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
