# Proposal: GET Endpoint `/ctv/vast` for the CTV VAST Enrichment Module

## Current State

The `ctv_vast_enrichment` module currently works exclusively as a **PBS hook** at the `RawBidderResponse` stage — it enriches VAST XML in bidder responses during a standard POST `/openrtb2/auction` auction.

A `handler.go` file already exists with a prepared `Handler struct` implementing `http.Handler` (via the `ServeHTTP` method), but **there is no mechanism to register** this handler in the PBS router. The handler is "orphaned" — it is not attached to any HTTP route.

### Core Problem

The PBS module system (`modules.NewBuilder().Build()`) only returns a `hooks.HookRepository` — a hook repository. **Modules have no access to the HTTP router** and cannot register endpoints on their own. All HTTP routes are hardcoded in `router/router.go` within the `New()` function.

## Proposed Approaches

### Option A: Register endpoint directly in the router (recommended)

Same approach used by existing endpoints (`/openrtb2/amp`, `/openrtb2/video`, `/event`, etc.) — the endpoint is created in `router/router.go` and registered manually.

#### Implementation Steps

**1. New `ModuleEndpointProvider` interface (optional, but cleaner)**

To avoid hardcoding everything in the router, the module can expose an interface that declares its endpoints:

```go
// modules/moduledeps/endpoint.go
package moduledeps

import "net/http"

// ModuleEndpoint describes an HTTP endpoint provided by a module.
type ModuleEndpoint struct {
    Method  string       // "GET", "POST"
    Path    string       // e.g. "/ctv/vast"
    Handler http.Handler
}

// EndpointProvider is an optional interface that a module can implement
// to register its own HTTP endpoints.
type EndpointProvider interface {
    Endpoints() []ModuleEndpoint
}
```

**2. Implement the interface in the module**

```go
// modules/prebid/ctv_vast_enrichment/module.go

func (m Module) Endpoints() []moduledeps.ModuleEndpoint {
    handler := NewHandler().
        WithConfig(configToReceiverConfig(MergeCTVVastConfig(&m.hostConfig, nil, nil)))
    // Selector, Enricher, Formatter will be injected by the router
    // or wired with default implementations

    return []moduledeps.ModuleEndpoint{
        {
            Method:  "GET",
            Path:    "/ctv/vast",
            Handler: handler,
        },
    }
}
```

**3. Registration in the router**

Extend `router/router.go` — after `Build()` of modules, check if modules implement `EndpointProvider`:

```go
// router/router.go in the New() function, after modules.NewBuilder().Build()

// Register module endpoints
for id, module := range builtModules {
    if ep, ok := module.(moduledeps.EndpointProvider); ok {
        for _, endpoint := range ep.Endpoints() {
            logger.Infof("Registering module endpoint: %s %s (module: %s)", endpoint.Method, endpoint.Path, id)
            r.Handler(endpoint.Method, endpoint.Path, endpoint.Handler)
        }
    }
}
```

**4. Inject AuctionFunc**

The handler needs `AuctionFunc` to invoke auctions. This is the most critical element — a reference to the exchange must be passed:

```go
handler.WithAuctionFunc(func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
    // Use theExchange.HoldAuction() or a dedicated method
    return theExchange.RunSimpleAuction(ctx, req)
})
```

#### Flow Diagram (Option A)

```
GET /ctv/vast?pod_id=123&duration=30&max_ads=3
        │
        ▼
┌─────────────────────┐
│  router/router.go   │  ← registration: r.Handler("GET", "/ctv/vast", handler)
│  httprouter         │
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│  Handler.ServeHTTP  │  ← handler.go (already exists)
│                     │
│  1. Parse query     │  ← buildBidRequest() - TODO to implement
│  2. Build BidReq    │
│  3. AuctionFunc()   │  ← injected exchange
│  4. Pipeline VAST   │  ← BuildVastFromBidResponse() - already exists
│  5. Return XML      │
└─────────────────────┘
```

---

### Option B: Endpoint as a separate package in `endpoints/` (simpler, no new interface)

Without creating a new module interface — simply add the endpoint in `endpoints/` like the others:

#### Steps

**1. New file `endpoints/ctv_vast.go`**

```go
package endpoints

import (
    "net/http"

    "github.com/julienschmidt/httprouter"
    ctv "github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment"
)

func NewCTVVastEndpoint(cfg ctv.ReceiverConfig, auctionFn ctv.AuctionFunc) httprouter.Handle {
    handler := ctv.NewHandler().
        WithConfig(cfg).
        WithSelector(selectpkg.NewDefaultSelector()).
        WithEnricher(enrichpkg.NewDefaultEnricher()).
        WithFormatter(formatpkg.NewGAMSSUFormatter()).
        WithAuctionFunc(auctionFn)

    return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
        handler.ServeHTTP(w, r)
    }
}
```

**2. Registration in `router/router.go`**

```go
// In the New() function, alongside existing endpoints:
if cfg.Hooks.Modules["prebid"]["ctv_vast_enrichment"] != nil {
    r.GET("/ctv/vast", endpoints.NewCTVVastEndpoint(ctvConfig, auctionFunc))
}
```

---

### Option C: Registration via Exitpoint Hook (no router changes)

Instead of a new endpoint, the module could "hijack" the existing `/openrtb2/auction` endpoint via the `Exitpoint` hook and change the response format to VAST XML when it detects a special parameter. **Not recommended** — this is a hack, not a clean solution.

---

## Recommendation

**Option A** (the `EndpointProvider` interface) is the cleanest architecturally:

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| Architectural cleanliness | ✅ Extensible | ⚠️ Hardcoded | ❌ Hack |
| Ease of implementation | ⚠️ New interface | ✅ Simple | ✅ Simple |
| Reusability | ✅ Other modules benefit too | ❌ One-off | ❌ One-off |
| PBS compatibility | ⚠️ Requires changes in `modules/` | ✅ Existing pattern | ⚠️ Hook misuse |
| Minimum core changes | ❌ 3 core files | ✅ 2 core files | ✅ 0 core files |

**For a quick MVP: Option B** — fewest changes, follows existing PBS patterns.

**For long-term architecture: Option A** — creates a reusable mechanism for future modules.

---

## Work Required Regardless of Approach

No matter which option is chosen, `handler.go` needs implementation:

### 1. Query parameter parsing (`buildBidRequest`)

```go
func (h *Handler) buildBidRequest(r *http.Request) (*openrtb2.BidRequest, error) {
    q := r.URL.Query()

    podID := q.Get("pod_id")           // required
    duration := q.Get("duration")       // max duration in seconds
    maxAds := q.Get("max_ads")          // max ads in pod
    publisherID := q.Get("pub_id")      // publisher ID
    siteURL := q.Get("url")             // site URL
    // ... additional parameters
}
```

### 2. Inject Selector/Enricher/Formatter

The handler needs concrete implementations of the interfaces. Decisions to make:
- Create them in the module Builder?
- Create them in the endpoint within the router?

### 3. Inject AuctionFunc

The most critical dependency — the handler must be able to invoke a PBS auction. Requires access to `exchange.Exchange`.

### 4. Tests

- Unit test `handler_test.go` with mocked `AuctionFunc`
- Integration test with full pipeline (Postman collection partially exists)

---

## Implementation Path Summary (Option B — MVP)

```
1. endpoints/ctv_vast.go          ← new file: endpoint wrapper
2. router/router.go               ← add r.GET("/ctv/vast", ...)
3. handler.go                     ← implement buildBidRequest()
4. handler_test.go                ← tests
5. config (optional)              ← feature flag in pbs.json
```

Total: ~5 files to change/create.

---

## How Modules Are Triggered by a GET Endpoint — Hook Executor Mechanism

Modules are **not triggered "by the endpoint" directly** — they are triggered by the **hook executor**, which the endpoint creates and invokes at the appropriate stages.

### Call Chain (AMP Example)

```
GET /openrtb2/amp?tag_id=xyz
  │
  ▼
router.go: r.GET("/openrtb2/amp", ampEndpoint)
  │
  ▼
AmpAuction handler creates hook executor:
  hookExecutor := hookexecution.NewHookExecutor(planBuilder, "/openrtb2/amp", metrics)
  │
  ├─► hookExecutor.ExecuteEntrypointStage(r, nil)     ← modules get raw *http.Request
  │
  ├─► exchange.HoldAuction(ctx, AuctionRequest{HookExecutor: hookExecutor, ...})
  │     inside exchange:
  │     ├─► ExecuteProcessedAuctionStage()
  │     ├─► ExecuteBidderRequestStage()        ← per bidder
  │     ├─► ExecuteRawBidderResponseStage()    ← ctv_vast_enrichment fires HERE!
  │     └─► ExecuteAllProcessedBidResponsesStage()
  │
  ├─► hookExecutor.ExecuteAuctionResponseStage(response)
  └─► hookExecutor.ExecuteExitpointStage(ampResponse, w)
```

AMP runs **7 of 8 stages** — all except `RawAuctionRequest` (because GET has no JSON body).

### Key Mechanism: Execution Plan

The hook executor knows which modules to run because it **looks up the endpoint in the `host_execution_plan`** — a literal map lookup (`hooks/plan.go`):

```go
cfg.Endpoints[endpoint].Stages[stage].Groups
```

If the endpoint **is not a key** in the `endpoints` map — **no hooks will fire**, even if the module implements the interface. Configuration structure (`config/hooks.go`):

```go
type HookExecutionPlan struct {
    Endpoints map[string]struct {      // key = "/openrtb2/auction", "/openrtb2/amp", etc.
        Stages map[string]struct {     // key = "entrypoint", "raw_bidder_response", etc.
            Groups []HookExecutionGroup
        }
    }
}
```

### What This Means for a New `/ctv/vast` Endpoint

The new GET endpoint must do **exactly what AMP does**:

**1. Define an endpoint constant** in `hooks/hookexecution/executor.go`:

```go
const EndpointCtvVast = "/ctv/vast"
```

**2. Create a hook executor** in the handler:

```go
hookExecutor := hookexecution.NewHookExecutor(planBuilder, EndpointCtvVast, metrics)
```

**3. Call hook stages** at the appropriate points:

```go
// At the start
hookExecutor.ExecuteEntrypointStage(r, nil)

// Pass executor to exchange
exchange.HoldAuction(ctx, AuctionRequest{HookExecutor: hookExecutor, ...})
// ↑ inside exchange, these fire automatically:
//   ProcessedAuctionRequest, BidderRequest, RawBidderResponse, AllProcessedBidResponses

// After auction
hookExecutor.ExecuteAuctionResponseStage(response)
hookExecutor.ExecuteExitpointStage(vastXML, w)
```

**4. Add the endpoint to the execution plan** in `pbs.json`:

```json
"host_execution_plan": {
  "endpoints": {
    "/openrtb2/auction": { ... },
    "/ctv/vast": {
      "stages": {
        "raw_bidder_response": {
          "groups": [
            {
              "timeout": 1000,
              "hook_sequence": [
                {
                  "module_code": "prebid.ctv_vast_enrichment",
                  "hook_impl_code": "code123"
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

### Mechanism Summary

| Element | Role |
|---------|------|
| `hookexecution.NewHookExecutor(planBuilder, endpoint, metrics)` | Creates executor bound to the endpoint |
| `planBuilder.PlanForXxxStage(endpoint)` | Looks up hooks in execution plan for the given endpoint |
| `host_execution_plan.endpoints["/ctv/vast"]` | **Configuration decides** which modules fire |
| `exchange.HoldAuction(req{HookExecutor})` | Internally fires BidderRequest, RawBidderResponse, etc. |

The `ctv_vast_enrichment` module will work **automatically** on the new GET endpoint — as long as:
- the handler creates a `hookExecutor` and passes it to `exchange.HoldAuction`
- the endpoint is added to the `host_execution_plan` in configuration

**No new interface is needed to trigger hooks** — the entire mechanism already exists. All that's needed is to wire the handler into the router and give it access to `exchange` + `planBuilder`.

### Updated Implementation Path

```
1. hooks/hookexecution/executor.go  ← new EndpointCtvVast constant
2. endpoints/ctv_vast.go            ← new handler with hookExecutor
3. router/router.go                 ← r.GET("/ctv/vast", ...)
4. handler.go                       ← implement buildBidRequest()
5. pbs.json                         ← add /ctv/vast to execution plan
6. handler_test.go                  ← tests
```
