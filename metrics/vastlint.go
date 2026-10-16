package metrics

// VastlintEvent is a bounded collection counter. Values never contain auction data.
type VastlintEvent string

func VastlintEvents() []VastlintEvent {
	return []VastlintEvent{"auctions", "sampled", "scan_limited", "non_video", "missing_markup", "url_markup", "oversized_markup", "oversized_extension", "capacity_dropped", "rate_dropped", "expired", "submitted", "checked", "with_findings", "failed", "shutdown_dropped", "account_excluded", "response_excluded", "privacy_suppressed", "activity_denied", "sample_excluded", "candidates", "no_candidate", "timeout", "transport_failed", "incompatible_response", "worker_panicked", "with_errors", "with_warnings", "with_infos", "unsupported"}
}

func (event VastlintEvent) Valid() bool {
	switch event {
	case "auctions", "sampled", "scan_limited", "non_video", "missing_markup", "url_markup", "oversized_markup", "oversized_extension", "capacity_dropped", "rate_dropped", "expired", "submitted", "checked", "with_findings", "failed", "shutdown_dropped", "account_excluded", "response_excluded", "privacy_suppressed", "activity_denied", "sample_excluded", "candidates", "no_candidate", "timeout", "transport_failed", "incompatible_response", "worker_panicked", "with_errors", "with_warnings", "with_infos", "unsupported":
		return true
	}
	return false
}
