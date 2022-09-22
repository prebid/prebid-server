package prometheusmetrics

import (
	"strconv"
	"time"

	"github.com/prebid/prebid-server/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// RecordAdapterDuplicateBidID captures the  bid.ID collisions when adaptor
// gives the bid response with multiple bids containing  same bid.ID
// ensure collisions value is greater than 1. This function will not give any error
// if collisions = 1 is passed
func (m *Metrics) RecordAdapterDuplicateBidID(adaptor string, collisions int) {
	m.adapterDuplicateBidIDCounter.With(prometheus.Labels{
		adapterLabel: adaptor,
	}).Add(float64(collisions))
}

// RecordRequestHavingDuplicateBidID keeps count of request when duplicate bid.id is
// detected in partner's response
func (m *Metrics) RecordRequestHavingDuplicateBidID() {
	m.requestsDuplicateBidIDCounter.Inc()
}

// pod specific metrics

// recordAlgoTime is common method which handles algorithm time performance
func recordAlgoTime(timer *prometheus.HistogramVec, labels metrics.PodLabels, elapsedTime time.Duration) {

	pmLabels := prometheus.Labels{
		podAlgorithm: labels.AlgorithmName,
	}

	if labels.NoOfImpressions != nil {
		pmLabels[podNoOfImpressions] = strconv.Itoa(*labels.NoOfImpressions)
	}
	if labels.NoOfCombinations != nil {
		pmLabels[podTotalCombinations] = strconv.Itoa(*labels.NoOfCombinations)
	}
	if labels.NoOfResponseBids != nil {
		pmLabels[podNoOfResponseBids] = strconv.Itoa(*labels.NoOfResponseBids)
	}

	timer.With(pmLabels).Observe(elapsedTime.Seconds())
}

// RecordPodImpGenTime records number of impressions generated and time taken
// by underneath algorithm to generate them
func (m *Metrics) RecordPodImpGenTime(labels metrics.PodLabels, start time.Time) {
	elapsedTime := time.Since(start)
	recordAlgoTime(m.podImpGenTimer, labels, elapsedTime)
}

// RecordPodCombGenTime records number of combinations generated and time taken
// by underneath algorithm to generate them
func (m *Metrics) RecordPodCombGenTime(labels metrics.PodLabels, elapsedTime time.Duration) {
	recordAlgoTime(m.podCombGenTimer, labels, elapsedTime)
}

// RecordPodCompititveExclusionTime records number of combinations comsumed for forming
// final ad pod response and time taken by underneath algorithm to generate them
func (m *Metrics) RecordPodCompititveExclusionTime(labels metrics.PodLabels, elapsedTime time.Duration) {
	recordAlgoTime(m.podCompExclTimer, labels, elapsedTime)
}

//RecordAdapterVideoBidDuration records actual ad duration (>0) returned by the bidder
func (m *Metrics) RecordAdapterVideoBidDuration(labels metrics.AdapterLabels, videoBidDuration int) {
	if videoBidDuration > 0 {
		m.adapterVideoBidDuration.With(prometheus.Labels{adapterLabel: string(labels.Adapter)}).Observe(float64(videoBidDuration))
	}
}
