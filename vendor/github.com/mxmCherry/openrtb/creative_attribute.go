package openrtb

// 5.3 Creative Attributes
//
// Standard list of creative attributes that can describe an ad being served or serve as restrictions of thereof.
type CreativeAttribute int8

const (
	CreativeAttributeAudioAdAutoPlay                                CreativeAttribute = 1  // Audio Ad (Auto-Play)
	CreativeAttributeAudioAdUserInitiated                           CreativeAttribute = 2  // Audio Ad (User Initiated)
	CreativeAttributeExpandableAutomatic                            CreativeAttribute = 3  // Expandable (Automatic)
	CreativeAttributeExpandableUserInitiatedClick                   CreativeAttribute = 4  // Expandable (User Initiated - Click)
	CreativeAttributeExpandableUserInitiatedRollover                CreativeAttribute = 5  // Expandable (User Initiated - Rollover)
	CreativeAttributeInBannerVideoAdAutoPlay                        CreativeAttribute = 6  // In-Banner Video Ad (Auto-Play)
	CreativeAttributeInBannerVideoAdUserInitiated                   CreativeAttribute = 7  // In-Banner Video Ad (User Initiated)
	CreativeAttributePop                                            CreativeAttribute = 8  // Pop (e.g., Over, Under, or Upon Exit)
	CreativeAttributeProvocativeOrSuggestiveImagery                 CreativeAttribute = 9  // Provocative or Suggestive Imagery
	CreativeAttributeShakyFlashingFlickeringExtremeAnimationSmileys CreativeAttribute = 10 // Shaky, Flashing, Flickering, Extreme Animation, Smileys
	CreativeAttributeSurveys                                        CreativeAttribute = 11 // Surveys
	CreativeAttributeTextOnly                                       CreativeAttribute = 12 // Text Only
	CreativeAttributeUserInteractive                                CreativeAttribute = 13 // User Interactive (e.g., Embedded Games)
	CreativeAttributeWindowsDialogOrAlertStyle                      CreativeAttribute = 14 // Windows Dialog or Alert Style
	CreativeAttributeHasAudioOnOffButton                            CreativeAttribute = 15 // Has Audio On/Off Button
	CreativeAttributeAdProvidesSkipButton                           CreativeAttribute = 16 // Ad Provides Skip Button (e.g. VPAID-rendered skip button on pre-roll video)
	CreativeAttributeAdobeFlash                                     CreativeAttribute = 17 // Adobe Flash
)
