package openrtb2

import "encoding/json"

// 3.2.5 Object: Metric
//
// This object is associated with an impression as an array of metrics.
// These metrics can offer insight into the impression to assist with decisioning such as average recent viewability, click-through rate, etc.
// Each metric is identified by its type, reports the value of the metric, and optionally identifies the source or vendor measuring the value.
type Metric struct {

	// Attribute:
	//   type
	// Type:
	//   string; required
	// Description:
	//   Type of metric being presented using exchange curated string
	//   names which should be published to bidders a priori.\
	Type string `json:"type"`

	// Attribute:
	//   value
	// Type:
	//   float; required
	// Dscription:
	//   Number representing the value of the metric. Probabilities
	//   must be in the range 0.0 – 1.0.
	Value float64 `json:"value,omitempty"`

	// Attribute:
	//   vendor
	// Type:
	//   string; recommended
	// Description:
	//   Source of the value using exchange curated string names
	//   which should be published to bidders a priori. If the exchange
	//   itself is the source versus a third party, “EXCHANGE” is
	//   recommended.
	Vendor string `json:"vendor,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
