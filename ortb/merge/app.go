package merge

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

func App(v *openrtb2.App, overrideJSON json.RawMessage) error {
	*v = *ortb.CloneApp(v)

	// Track EXTs
	// It's not necessary to track `ext` fields in array items because the array
	// items will be replaced entirely with the override JSON, so no merge is required.
	var ext, extPublisher, extContent, extContentProducer, extContentNetwork, extContentChannel extMerger
	ext.Track(&v.Ext)
	if v.Publisher != nil {
		extPublisher.Track(&v.Publisher.Ext)
	}
	if v.Content != nil {
		extContent.Track(&v.Content.Ext)
	}
	if v.Content != nil && v.Content.Producer != nil {
		extContentProducer.Track(&v.Content.Producer.Ext)
	}
	if v.Content != nil && v.Content.Network != nil {
		extContentNetwork.Track(&v.Content.Network.Ext)
	}
	if v.Content != nil && v.Content.Channel != nil {
		extContentChannel.Track(&v.Content.Channel.Ext)
	}

	// Merge
	if err := jsonutil.Unmarshal(overrideJSON, &v); err != nil {
		return err
	}

	// Merge EXTs
	if err := ext.Merge(); err != nil {
		return err
	}
	if err := extPublisher.Merge(); err != nil {
		return err
	}
	if err := extContent.Merge(); err != nil {
		return err
	}
	if err := extContentProducer.Merge(); err != nil {
		return err
	}
	if err := extContentNetwork.Merge(); err != nil {
		return err
	}
	if err := extContentChannel.Merge(); err != nil {
		return err
	}

	return nil
}
