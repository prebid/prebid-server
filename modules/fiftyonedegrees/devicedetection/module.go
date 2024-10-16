package devicedetection

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/pkg/errors"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/prebid/prebid-server/v2/modules/moduledeps"
)

func configHashFromConfig(cfg *config) *dd.ConfigHash {
	configHash := dd.NewConfigHash(cfg.getPerformanceProfile())
	if cfg.Performance.Concurrency != nil {
		configHash.SetConcurrency(uint16(*cfg.Performance.Concurrency))
	}

	if cfg.Performance.Difference != nil {
		configHash.SetDifference(int32(*cfg.Performance.Difference))
	}

	if cfg.Performance.AllowUnmatched != nil {
		configHash.SetAllowUnmatched(*cfg.Performance.AllowUnmatched)
	}

	if cfg.Performance.Drift != nil {
		configHash.SetDrift(int32(*cfg.Performance.Drift))
	}
	return configHash
}

func Builder(rawConfig json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := parseConfig(rawConfig)
	if err != nil {
		return Module{}, errors.Wrap(err, "failed to parse config")
	}

	err = validateConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	configHash := configHashFromConfig(&cfg)

	deviceDetectorImpl, err := newDeviceDetector(
		configHash,
		&cfg,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create device detector")
	}

	return Module{
			cfg,
			deviceDetectorImpl,
			newEvidenceExtractor(),
			newAccountValidator(),
		},
		nil
}

type Module struct {
	config            config
	deviceDetector    deviceDetector
	evidenceExtractor evidenceExtractor
	accountValidator  accountValidator
}

type deviceDetector interface {
	getSupportedHeaders() []dd.EvidenceKey
	getDeviceInfo(evidence []onpremise.Evidence, ua string) (*deviceInfo, error)
}

type accountValidator interface {
	isAllowed(cfg config, req []byte) bool
}

type evidenceExtractor interface {
	fromHeaders(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []stringEvidence
	fromSuaPayload(payload []byte) []stringEvidence
	extract(ctx hookstage.ModuleContext) ([]onpremise.Evidence, string, error)
}

func (m Module) HandleEntrypointHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return handleAuctionEntryPointRequestHook(
		m.config,
		payload,
		m.deviceDetector,
		m.evidenceExtractor,
		m.accountValidator,
	)
}

func (m Module) HandleRawAuctionHook(
	_ context.Context,
	mCtx hookstage.ModuleInvocationContext,
	_ hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return handleAuctionRequestHook(mCtx, m.deviceDetector, m.evidenceExtractor)
}
