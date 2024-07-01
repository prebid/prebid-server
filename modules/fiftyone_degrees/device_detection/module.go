package device_detection

import (
	"context"
	"encoding/json"
	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/pkg/errors"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/prebid/prebid-server/v2/modules/moduledeps"
	"net/http"
)

func configHashFromConfig(cfg *Config) *dd.ConfigHash {
	configHash := dd.NewConfigHash(cfg.GetPerformanceProfile())
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
	var cfg Config
	ncfg, err := ParseConfig(rawConfig)
	if err != nil {
		return Module{}, errors.Wrap(err, "failed to parse config")
	}

	err = ValidateConfig(ncfg)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	cfg = ncfg

	configHash := configHashFromConfig(&cfg)

	deviceDetector, err := NewDeviceDetector(
		configHash,
		&cfg,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create device detector")
	}

	return Module{
			cfg,
			deviceDetector,
			NewEvidenceExtractor(),
			NewAccountValidator(),
		},
		nil
}

type Module struct {
	config            Config
	deviceDetector    deviceDetector
	evidenceExtractor evidenceExtractor
	accountValidator  accountValidator
}

type deviceDetector interface {
	GetSupportedHeaders() []dd.EvidenceKey
	GetDeviceInfo(evidence []onpremise.Evidence, ua string) (*DeviceInfo, error)
}

type accountValidator interface {
	IsWhiteListed(cfg Config, req []byte) bool
}

type evidenceExtractor interface {
	FromHeaders(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []StringEvidence
	FromSuaPayload(request *http.Request, payload []byte) []StringEvidence
	Extract(ctx hookstage.ModuleContext) ([]onpremise.Evidence, string, error)
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
