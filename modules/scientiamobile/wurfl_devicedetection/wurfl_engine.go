//go:build wurfl

package wurfl_devicedetection

import (
	"fmt"
	"strings"

	wurfl "github.com/WURFL/golang-wurfl"
)

// declare conformity with  wurflDeviceDetection interface
var _ wurflDeviceDetection = (*wurflEngine)(nil)

// newWurflEngine creates a new Enricher
func newWurflEngine(c config) (wurflDeviceDetection, error) {
	err := wurfl.Download(c.WURFLSnapshotURL, c.WURFLFileDirPath)
	if err != nil {
		return nil, err
	}
	wengine, err := wurfl.Create(c.WURFLFilePath(),
		nil,
		nil,
		-1,
		wurfl.WurflCacheProviderLru,
		c.WURFLEngineCacheSize(),
	)
	if err != nil {
		return nil, err
	}

	if c.WURFLRunUpdater == nil || *c.WURFLRunUpdater {
		uerr := wengine.SetUpdaterDataURL(c.WURFLSnapshotURL)
		if uerr != nil {
			return nil, fmt.Errorf("could not set WURFL Updater Snapshot URL: %sn", uerr.Error())
		}

		uerr = wengine.SetUpdaterDataFrequency(wurfl.WurflUpdaterFrequencyDaily)
		if uerr != nil {
			return nil, fmt.Errorf("could not set the WURFL Updater frequency: %s", uerr.Error())
		}

		uerr = wengine.UpdaterStart()
		if uerr != nil {
			return nil, fmt.Errorf("could not start the WURFL Updater: %s", uerr.Error())
		}

	}

	caps := wengine.GetAllCaps()
	e := &wurflEngine{
		wengine: wengine,
		caps:    caps,
		vcaps:   vcaps,
	}

	err = e.validate()
	if err != nil {
		return nil, err
	}

	return e, nil
}

// wurflEngine is the ortb2 enricher powered by WURFL
type wurflEngine struct {
	wengine *wurfl.Wurfl
	caps    []string
	vcaps   []string
}

// deviceDetection performs device detection using the WURFL engine.
func (e *wurflEngine) DeviceDetection(headers map[string]string) (wurflData, error) {
	wurflDevice, err := e.wengine.LookupWithImportantHeaderMap(headers)
	if err != nil {
		return nil, err
	}
	defer wurflDevice.Destroy()

	wurflDeviceID, err := wurflDevice.GetDeviceID()
	if err != nil {
		return nil, err
	}
	wurflData, err := wurflDevice.GetStaticCaps(e.caps)
	if err != nil {
		return nil, err
	}
	vcaps, err := wurflDevice.GetVirtualCaps(e.vcaps)
	if err != nil {
		return nil, err
	}
	for k, v := range vcaps {
		wurflData[k] = v
	}
	wurflData[wurflID] = wurflDeviceID
	return wurflData, nil
}

// validate checks if the WURFL file has all the required capabilities
func (e *wurflEngine) validate() error {
	requiredCaps := []string{
		ajaxSupportJavascriptCapKey,
		brandNameCapKey,
		densityClassCapKey,
		isConnectedTVCapKey,
		isOTTCapKey,
		isTabletCapKey,
		modelNameCapKey,
		physicalFormFactorCapKey,
		resolutionWidthCapKey,
		resolutionWidthCapKey,
	}
	m := map[string]bool{}
	for _, val := range e.caps {
		m[val] = true
	}
	missed := []string{}
	for _, val := range requiredCaps {
		if _, ok := m[val]; !ok {
			missed = append(missed, val)
		}
	}
	if len(missed) > 0 {
		return fmt.Errorf("WURFL file is missing the following capabilities: %s", strings.Join(missed, ","))
	}
	return nil
}
