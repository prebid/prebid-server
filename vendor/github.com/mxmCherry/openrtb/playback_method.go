package openrtb

// 5.10 Playback Methods
//
// Various playback methods.
type PlaybackMethod int8

const (
	PlaybackMethodPageLoadSoundOn          PlaybackMethod = 1 // Initiates on Page Load with Sound On
	PlaybackMethodPageLoadSoundOff         PlaybackMethod = 2 // Initiates on Page Load with Sound Off by Default
	PlaybackMethodClickSoundOn             PlaybackMethod = 3 // Initiates on Click with Sound On
	PlaybackMethodMouseOverSoundOn         PlaybackMethod = 4 // Initiates on Mouse-Over with Sound On
	PlaybackMethodEnteringViewportSoundOn  PlaybackMethod = 5 // Initiates on Entering Viewport with Sound On
	PlaybackMethodEnteringViewportSoundOff PlaybackMethod = 6 // Initiates on Entering Viewport with Sound Off by Default
)
