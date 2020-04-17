package openrtb

// 5.17 Volume Normalization Modes
//
// Types of volume normalization modes, typically for audio.
type VolumeNormalizationMode int8

const (
	VolumeNormalizationModeNone                               VolumeNormalizationMode = 0 // None
	VolumeNormalizationModeAdVolumeAverageNormalizedToContent VolumeNormalizationMode = 1 // Ad Volume Average Normalized to Content
	VolumeNormalizationModeAdVolumePeakNormalizedToContent    VolumeNormalizationMode = 2 // Ad Volume Peak Normalized to Content
	VolumeNormalizationModeAdLoudnessNormalizedToContent      VolumeNormalizationMode = 3 // Ad Loudness Normalized to Content
	VolumeNormalizationModeCustomVolumeNormalizationMode      VolumeNormalizationMode = 4 // Custom Volume Normalization
)

// Ptr returns pointer to own value.
func (m VolumeNormalizationMode) Ptr() *VolumeNormalizationMode {
	return &m
}

// Val safely dereferences pointer, returning default value (VolumeNormalizationModeNone) for nil.
func (m *VolumeNormalizationMode) Val() VolumeNormalizationMode {
	if m == nil {
		return VolumeNormalizationModeNone
	}
	return *m
}
