package metrics

import "time"

// RecordAdapterDuplicateBidID as noop
func (me *Metrics) RecordAdapterDuplicateBidID(adaptor string, collisions int) {
}

// RecordRequestHavingDuplicateBidID as noop
func (me *Metrics) RecordRequestHavingDuplicateBidID() {
}

// RecordPodImpGenTime as a noop
func (me *Metrics) RecordPodImpGenTime(labels PodLabels, startTime time.Time) {
}

// RecordPodCombGenTime as a noop
func (me *Metrics) RecordPodCombGenTime(labels PodLabels, elapsedTime time.Duration) {
}

// RecordPodCompititveExclusionTime as a noop
func (me *Metrics) RecordPodCompititveExclusionTime(labels PodLabels, elapsedTime time.Duration) {
}

// RecordAdapterVideoBidDuration as a noop
func (me *Metrics) RecordAdapterVideoBidDuration(labels AdapterLabels, videoBidDuration int) {
}
