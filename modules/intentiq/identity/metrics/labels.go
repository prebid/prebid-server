package metrics

const (
	ResultHit    = "hit"
	ResultMiss   = "miss"
	ResultStored = "stored"
	ResultError  = "error"
)

const (
	OpGet = "get"
	OpPut = "put"
)

// not_enriched_total reasons.
const (
	ReasonNoIDs       = "no_ids"
	ReasonNoIDsCached = "no_ids_cached"
	ReasonInProgress  = "in_progress"
	ReasonNoEndpoint  = "no_endpoint"
)
