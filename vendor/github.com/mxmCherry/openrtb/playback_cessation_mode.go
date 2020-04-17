package openrtb

// 5.11 Playback Cessation Modes
//
// Various modes for when playback terminates.
type PlaybackCessationMode int8

const (
	PlaybackCessationModeVideoCompletionOrTerminatedByUser                     PlaybackCessationMode = 1 // On Video Completion or when Terminated by User
	PlaybackCessationModeLeavingViewportOrTerminatedByUser                     PlaybackCessationMode = 2 // On Leaving Viewport or when Terminated by User
	PlaybackCessationModeLeavingViewportUntilVideoCompletionOrTerminatedByUser PlaybackCessationMode = 3 // On Leaving Viewport Continues as a Floating/Slider Unit until Video Completion or when Terminated by User
)
