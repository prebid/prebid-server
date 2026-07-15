// Package tmp implements a Prebid Server module that acts as an OpenRTB→TMP
// adapter and TMP router. Each auction is fanned out to one or more configured
// TMP providers (identity agent, context agent, or both); responses are joined
// locally and surfaced on the bid response.
//
// The wire types, signing primitives and URL canonicalization come from
// github.com/adcontextprotocol/adcp-go. Property resolution and OpenRTB
// mapping are implemented here — adcp-go does not provide them.
package tmp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Builder is the entry point Prebid Server uses to instantiate the module.
func Builder(raw json.RawMessage, deps moduledeps.ModuleDeps) (any, error) {
	var cfg Config
	if err := jsonutil.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("adcontextprotocol.tmp: unmarshal config: %w", err)
	}

	privKey, err := cfg.validated()
	if err != nil {
		return nil, fmt.Errorf("adcontextprotocol.tmp: invalid config: %w", err)
	}

	signer, err := tmproto.NewSigner(cfg.Signing.KeyID, privKey)
	if err != nil {
		return nil, fmt.Errorf("adcontextprotocol.tmp: signer: %w", err)
	}

	httpClient := &http.Client{
		Timeout:   time.Duration(cfg.TimeoutMs) * time.Millisecond,
		Transport: deps.HTTPClient.Transport,
	}

	return &Module{
		cfg:      cfg,
		signer:   signer,
		http:     httpClient,
		registry: newPropertyResolver(cfg.PropertyRegistry, deps.HTTPClient.Transport),
	}, nil
}

// Module is the running module instance.
type Module struct {
	cfg      Config
	signer   *tmproto.Signer
	http     *http.Client
	registry *propertyResolver
}

// asyncKey names the entry we stash the in-flight request under on the module
// invocation context.
const asyncKey = "adcontextprotocol.tmp.asyncRequest"

// Hook interface assertions — the compiler catches signature drift here.
var (
	_ hookstage.Entrypoint              = (*Module)(nil)
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

// asyncRequest carries a single auction's in-flight TMP fan-out from the
// entrypoint hook through to the response hook.
type asyncRequest struct {
	done   chan struct{}
	result *routerResult
	err    error
}
