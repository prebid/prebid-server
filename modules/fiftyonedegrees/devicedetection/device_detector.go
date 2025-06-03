package devicedetection

import (
	"fmt"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
)

type engine interface {
	Process(evidences []onpremise.Evidence) (*dd.ResultsHash, error)
	GetHttpHeaderKeys() []dd.EvidenceKey
}

type extractor interface {
	extract(results Results, ua string) (*deviceInfo, error)
}

type defaultDeviceDetector struct {
	cfg                 *dd.ConfigHash
	deviceInfoExtractor extractor
	engine              engine
}

func newDeviceDetector(cfg *dd.ConfigHash, moduleConfig *config) (*defaultDeviceDetector, error) {
	engineOptions := buildEngineOptions(moduleConfig, cfg)

	ddEngine, err := onpremise.New(
		engineOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create onpremise engine: %w", err)
	}

	deviceDetector := &defaultDeviceDetector{
		engine:              ddEngine,
		cfg:                 cfg,
		deviceInfoExtractor: newDeviceInfoExtractor(),
	}

	return deviceDetector, nil
}

func buildEngineOptions(moduleConfig *config, configHash *dd.ConfigHash) []onpremise.EngineOptions {
	options := []onpremise.EngineOptions{
		onpremise.WithDataFile(moduleConfig.DataFile.Path),
	}

	options = append(
		options,
		onpremise.WithProperties([]string{
			"HardwareVendor",
			"HardwareName",
			"DeviceType",
			"PlatformVendor",
			"PlatformName",
			"PlatformVersion",
			"BrowserVendor",
			"BrowserName",
			"BrowserVersion",
			"ScreenPixelsWidth",
			"ScreenPixelsHeight",
			"PixelRatio",
			"Javascript",
			"GeoLocation",
			"HardwareModel",
			"HardwareFamily",
			"HardwareModelVariants",
			"ScreenInchesHeight",
			"IsCrawler",
		}),
	)

	options = append(
		options,
		onpremise.WithConfigHash(configHash),
	)

	if moduleConfig.DataFile.MakeTempCopy != nil {
		options = append(
			options,
			onpremise.WithTempDataCopy(*moduleConfig.DataFile.MakeTempCopy),
		)
	}

	dataUpdateOptions := []onpremise.EngineOptions{
		onpremise.WithAutoUpdate(moduleConfig.DataFile.Update.Auto),
	}

	if moduleConfig.DataFile.Update.Url != "" {
		dataUpdateOptions = append(
			dataUpdateOptions,
			onpremise.WithDataUpdateUrl(
				moduleConfig.DataFile.Update.Url,
			),
		)
	}

	if moduleConfig.DataFile.Update.PollingInterval > 0 {
		dataUpdateOptions = append(
			dataUpdateOptions,
			onpremise.WithPollingInterval(
				moduleConfig.DataFile.Update.PollingInterval,
			),
		)
	}

	if moduleConfig.DataFile.Update.License != "" {
		dataUpdateOptions = append(
			dataUpdateOptions,
			onpremise.WithLicenseKey(moduleConfig.DataFile.Update.License),
		)
	}

	if moduleConfig.DataFile.Update.Product != "" {
		dataUpdateOptions = append(
			dataUpdateOptions,
			onpremise.WithProduct(moduleConfig.DataFile.Update.Product),
		)
	}

	if moduleConfig.DataFile.Update.WatchFileSystem != nil {
		dataUpdateOptions = append(
			dataUpdateOptions,
			onpremise.WithFileWatch(
				*moduleConfig.DataFile.Update.WatchFileSystem,
			),
		)
	}

	dataUpdateOptions = append(
		dataUpdateOptions,
		onpremise.WithUpdateOnStart(moduleConfig.DataFile.Update.OnStartup),
	)

	options = append(
		options,
		dataUpdateOptions...,
	)

	return options
}

func (x defaultDeviceDetector) getSupportedHeaders() []dd.EvidenceKey {
	return x.engine.GetHttpHeaderKeys()
}

func (x defaultDeviceDetector) getDeviceInfo(evidence []onpremise.Evidence, ua string) (*deviceInfo, error) {
	results, err := x.engine.Process(evidence)
	if err != nil {
		return nil, fmt.Errorf("failed to process evidence: %w", err)
	}
	defer results.Free()

	deviceInfo, err := x.deviceInfoExtractor.extract(results, ua)

	return deviceInfo, err
}
