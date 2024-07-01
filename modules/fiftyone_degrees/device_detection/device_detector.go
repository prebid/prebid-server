package device_detection

import (
	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/pkg/errors"
)

type engine interface {
	Process(evidences []onpremise.Evidence) (*dd.ResultsHash, error)
	GetHttpHeaderKeys() []dd.EvidenceKey
}

type extractor interface {
	Extract(results Results, ua string) (*DeviceInfo, error)
}

type DeviceDetector struct {
	cfg                 *dd.ConfigHash
	deviceInfoExtractor extractor
	engine              engine
}

func NewDeviceDetector(
	cfg *dd.ConfigHash,
	moduleConfig *Config,
) (*DeviceDetector, error) {
	engineOptions := buildEngineOptions(moduleConfig, cfg)

	ddEngine, err := onpremise.New(
		engineOptions...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create onpremise engine.")
	}

	deviceDetector := &DeviceDetector{
		engine:              ddEngine,
		cfg:                 cfg,
		deviceInfoExtractor: NewDeviceInfoExtractor(),
	}

	return deviceDetector, nil
}

func buildEngineOptions(moduleConfig *Config, configHash *dd.ConfigHash) []onpremise.EngineOptions {
	options := []onpremise.EngineOptions{
		onpremise.WithDataFile(moduleConfig.DataFile.Path),
	}

	options = append(
		options,
		onpremise.WithProperties([]string{"HardwareVendor",
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

	dataUpdateOptions := []onpremise.EngineOptions{}

	dataUpdateOptions = append(
		dataUpdateOptions,
		onpremise.WithAutoUpdate(moduleConfig.DataFile.Update.Auto),
	)

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

func (x DeviceDetector) GetSupportedHeaders() []dd.EvidenceKey {
	return x.engine.GetHttpHeaderKeys()
}

func (x DeviceDetector) GetDeviceInfo(evidence []onpremise.Evidence, ua string) (*DeviceInfo, error) {
	results, err := x.engine.Process(evidence)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to process evidence")
	}
	defer results.Free()

	deviceInfo, err := x.deviceInfoExtractor.Extract(results, ua)

	return deviceInfo, err
}
