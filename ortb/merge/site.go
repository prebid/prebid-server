package merge

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

func Site(v *openrtb2.Site, overrideJSON json.RawMessage, bidderName string) (*openrtb2.Site, error) {
	c := ortb.CloneSite(v)

	// Track EXTs
	// It's not necessary to track `ext` fields in array items because the array
	// items will be replaced entirely with the override JSON, so no merge is required.
	var ext, extPublisher, extContent, extContentProducer, extContentNetwork, extContentChannel extMerger
	ext.Track(&c.Ext)
	if c.Publisher != nil {
		extPublisher.Track(&c.Publisher.Ext)
	}
	if c.Content != nil {
		extContent.Track(&c.Content.Ext)
	}
	if c.Content != nil && c.Content.Producer != nil {
		extContentProducer.Track(&c.Content.Producer.Ext)
	}
	if c.Content != nil && c.Content.Network != nil {
		extContentNetwork.Track(&c.Content.Network.Ext)
	}
	if c.Content != nil && c.Content.Channel != nil {
		extContentChannel.Track(&c.Content.Channel.Ext)
	}

	// Merge
	if err := jsonutil.Unmarshal(overrideJSON, &c); err != nil {
		return nil, err
	}

	// Merge EXTs
	if err := ext.Merge(); err != nil {
		return nil, err
	}
	if err := extPublisher.Merge(); err != nil {
		return nil, err
	}
	if err := extContent.Merge(); err != nil {
		return nil, err
	}
	if err := extContentProducer.Merge(); err != nil {
		return nil, err
	}
	if err := extContentNetwork.Merge(); err != nil {
		return nil, err
	}
	if err := extContentChannel.Merge(); err != nil {
		return nil, err
	}

	return c, nil
}
