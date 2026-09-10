package enrich

import (
	vast "github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment"
)

func init() {
	// Register VastEnricher as the default enricher for the module hook.
	// This breaks the potential import cycle: parent cannot import enrich directly,
	// so enrich registers itself via EnricherFactory when its package is loaded.
	// The modules/builder.go blank-imports this package to trigger the init.
	vast.EnricherFactory = func() vast.Enricher {
		return NewEnricher()
	}
}
