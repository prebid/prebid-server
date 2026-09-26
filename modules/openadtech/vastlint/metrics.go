package vastlint

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsKey is the stage-map key for this module: vendor.module with dots
// replaced, matching modules.moduleReplacer.
const MetricsKey = "openadtech_vastlint"

type recorder interface {
	Finding(caller, ruleID string, revenue bool)
}

type nopRecorder struct{}

func (nopRecorder) Finding(string, string, bool) {}

type promRecorder struct {
	findings *prometheus.CounterVec
}

func (p promRecorder) Finding(caller, ruleID string, revenue bool) {
	impact := "false"
	if revenue {
		impact = "true"
	}
	p.findings.WithLabelValues(caller, ruleID, impact).Inc()
}

var (
	tallyMu sync.RWMutex
	tally   recorder = nopRecorder{}
)

// Register adds vastlint_findings_total to the Prebid Server Prometheus
// registry. Hosts scrape it on the existing /metrics port. The series is
// created only when this module is enabled.
func Register(reg prometheus.Registerer, namespace, subsystem string) {
	if reg == nil {
		return
	}
	findings := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "vastlint_findings_total",
		Help:      "Count of vastlint revenue-impact findings on video adm, labeled by bidder and rule id.",
	}, []string{"caller", "rule_id", "revenue_impact"})
	reg.MustRegister(findings)

	tallyMu.Lock()
	tally = promRecorder{findings: findings}
	tallyMu.Unlock()
}

func currentRecorder() recorder {
	tallyMu.RLock()
	defer tallyMu.RUnlock()
	return tally
}

func resetRecorder() {
	tallyMu.Lock()
	tally = nopRecorder{}
	tallyMu.Unlock()
}
