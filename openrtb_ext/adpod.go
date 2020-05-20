package openrtb_ext

import (
	"errors"
	"strings"
)

var (
	errInvalidAdPodMinDuration                    = errors.New("imp.video.minduration must be number positive number")
	errInvalidAdPodMaxDuration                    = errors.New("imp.video.maxduration must be number positive non zero number")
	errInvalidAdPodDuration                       = errors.New("imp.video.minduration must be less than imp.video.maxduration")
	errInvalidMinDurationRange                    = errors.New("imp.video.ext.adpod.adminduration * imp.video.ext.adpod.minads should be greater than or equal to imp.video.minduration")
	errInvalidMaxDurationRange                    = errors.New("imp.video.ext.adpod.admaxduration * imp.video.ext.adpod.maxads should be less than or equal to imp.video.maxduration + imp.video.maxextended")
	errInvalidCrossPodAdvertiserExclusionPercent  = errors.New("request.ext.adpod.crosspodexcladv must be a number between 0 and 100")
	errInvalidCrossPodIABCategoryExclusionPercent = errors.New("request.ext.adpod.crosspodexcliabcat must be a number between 0 and 100")
	errInvalidIABCategoryExclusionWindow          = errors.New("request.ext.adpod.excliabcatwindow must be postive number")
	errInvalidAdvertiserExclusionWindow           = errors.New("request.ext.adpod.excladvwindow must be postive number")
	errInvalidAdPodOffset                         = errors.New("request.imp.video.ext.offset must be postive number")
	errInvalidMinAds                              = errors.New("%key%.ext.adpod.minads must be positive number")
	errInvalidMaxAds                              = errors.New("%key%.ext.adpod.maxads must be positive number")
	errInvalidMinDuration                         = errors.New("%key%.ext.adpod.adminduration must be positive number")
	errInvalidMaxDuration                         = errors.New("%key%.ext.adpod.admaxduration must be positive number")
	errInvalidAdvertiserExclusionPercent          = errors.New("%key%.ext.adpod.excladv must be number between 0 and 100")
	errInvalidIABCategoryExclusionPercent         = errors.New("%key%.ext.adpod.excliabcat must be number between 0 and 100")
	errInvalidMinMaxAds                           = errors.New("%key%.ext.adpod.minads must be less than %key%.ext.adpod.maxads")
	errInvalidMinMaxDuration                      = errors.New("%key%.ext.adpod.adminduration must be less than %key%.ext.adpod.admaxduration")
)

// ExtCTVBid defines the contract for bidresponse.seatbid.bid[i].ext
type ExtCTVBid struct {
	ExtBid
	AdPod *BidAdPodExt `json:"adpod,omitempty"`
}

// BidAdPodExt defines the prebid adpod response in bidresponse.ext.adpod parameter
type BidAdPodExt struct {
	RefBids []string `json:"refbids,omitempty"`
}

// ExtCTVRequest defines the contract for bidrequest.ext
type ExtCTVRequest struct {
	ExtRequest
	AdPod *ExtRequestAdPod `json:"adpod,omitempty"`
}

//ExtVideoAdPod structure to accept video specific more parameters like adpod
type ExtVideoAdPod struct {
	Offset *int        `json:"offset,omitempty"` // Minutes from start where this ad is intended to show
	AdPod  *VideoAdPod `json:"adpod,omitempty"`
}

//ExtRequestAdPod holds AdPod specific extension parameters at request level
type ExtRequestAdPod struct {
	VideoAdPod
	CrossPodAdvertiserExclusionPercent  *int `json:"crosspodexcladv,omitempty"`    //Percent Value - Across multiple impression there will be no ads from same advertiser. Note: These cross pod rule % values can not be more restrictive than per pod
	CrossPodIABCategoryExclusionPercent *int `json:"crosspodexcliabcat,omitempty"` //Percent Value - Across multiple impression there will be no ads from same advertiser
	IABCategoryExclusionWindow          *int `json:"excliabcatwindow,omitempty"`   //Duration in minute between pods where exclusive IAB rule needs to be applied
	AdvertiserExclusionWindow           *int `json:"excladvwindow,omitempty"`      //Duration in minute between pods where exclusive advertiser rule needs to be applied
}

//VideoAdPod holds Video AdPod specific extension parameters at impression level
type VideoAdPod struct {
	MinAds                      *int `json:"minads,omitempty"`        //Default 1 if not specified
	MaxAds                      *int `json:"maxads,omitempty"`        //Default 1 if not specified
	MinDuration                 *int `json:"adminduration,omitempty"` // (adpod.adminduration * adpod.minads) should be greater than or equal to video.minduration
	MaxDuration                 *int `json:"admaxduration,omitempty"` // (adpod.admaxduration * adpod.maxads) should be less than or equal to video.maxduration + video.maxextended
	AdvertiserExclusionPercent  *int `json:"excladv,omitempty"`       // Percent value 0 means none of the ads can be from same advertiser 100 means can have all same advertisers
	IABCategoryExclusionPercent *int `json:"excliabcat,omitempty"`    // Percent value 0 means all ads should be of different IAB categories.
}

/*
//UnmarshalJSON will unmarshal extension into ExtVideoAdPod object
func (ext *ExtVideoAdPod) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, ext)
}

//UnmarshalJSON will unmarshal extension into ExtRequestAdPod object
func (ext *ExtRequestAdPod) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, ext)
}
*/
//getRequestAdPodError will return request level error message
func getRequestAdPodError(err error) error {
	return errors.New(strings.Replace(err.Error(), "%key%", "req.ext", -1))
}

//getVideoAdPodError will return video adpod level error message
func getVideoAdPodError(err error) error {
	return errors.New(strings.Replace(err.Error(), "%key%", "imp.video.ext", -1))
}

func getIntPtr(v int) *int {
	return &v
}

//Validate will validate AdPod object
func (pod *VideoAdPod) Validate() (err []error) {
	if nil != pod.MinAds && *pod.MinAds <= 0 {
		err = append(err, errInvalidMinAds)
	}

	if nil != pod.MaxAds && *pod.MaxAds <= 0 {
		err = append(err, errInvalidMaxAds)
	}

	if nil != pod.MinDuration && *pod.MinDuration < 0 {
		err = append(err, errInvalidMinDuration)
	}

	if nil != pod.MaxDuration && *pod.MaxDuration < 0 {
		err = append(err, errInvalidMaxDuration)
	}

	if nil != pod.AdvertiserExclusionPercent && (*pod.AdvertiserExclusionPercent < 0 || *pod.AdvertiserExclusionPercent > 100) {
		err = append(err, errInvalidAdvertiserExclusionPercent)
	}

	if nil != pod.IABCategoryExclusionPercent && (*pod.IABCategoryExclusionPercent < 0 || *pod.IABCategoryExclusionPercent > 100) {
		err = append(err, errInvalidIABCategoryExclusionPercent)
	}

	if nil != pod.MinAds && nil != pod.MaxAds && *pod.MinAds > *pod.MaxAds {
		err = append(err, errInvalidMinMaxAds)
	}

	if nil != pod.MinDuration && nil != pod.MaxDuration && *pod.MinDuration > *pod.MaxDuration {
		err = append(err, errInvalidMinMaxDuration)
	}

	return
}

//Validate will validate ExtRequestAdPod object
func (ext *ExtRequestAdPod) Validate() (err []error) {
	if nil == ext {
		return
	}

	if nil != ext.CrossPodAdvertiserExclusionPercent &&
		(*ext.CrossPodAdvertiserExclusionPercent < 0 || *ext.CrossPodAdvertiserExclusionPercent > 100) {
		err = append(err, errInvalidCrossPodAdvertiserExclusionPercent)
	}

	if nil != ext.CrossPodIABCategoryExclusionPercent &&
		(*ext.CrossPodIABCategoryExclusionPercent < 0 || *ext.CrossPodIABCategoryExclusionPercent > 100) {
		err = append(err, errInvalidCrossPodIABCategoryExclusionPercent)
	}

	if nil != ext.IABCategoryExclusionWindow && *ext.IABCategoryExclusionWindow < 0 {
		err = append(err, errInvalidIABCategoryExclusionWindow)
	}

	if nil != ext.AdvertiserExclusionWindow && *ext.AdvertiserExclusionWindow < 0 {
		err = append(err, errInvalidAdvertiserExclusionWindow)
	}

	if errL := ext.VideoAdPod.Validate(); nil != errL {
		for _, errr := range errL {
			err = append(err, getRequestAdPodError(errr))
		}
	}

	return
}

//Validate will validate video extension object
func (ext *ExtVideoAdPod) Validate() (err []error) {
	if nil != ext.Offset && *ext.Offset < 0 {
		err = append(err, errInvalidAdPodOffset)
	}

	if nil != ext.AdPod {
		if errL := ext.AdPod.Validate(); nil != errL {
			for _, errr := range errL {
				err = append(err, getRequestAdPodError(errr))
			}
		}
	}

	return
}

//SetDefaultValue will set default values if not present
func (pod *VideoAdPod) SetDefaultValue() {
	//pod.MinAds setting default value
	if nil == pod.MinAds {
		pod.MinAds = getIntPtr(2)
	}

	//pod.MaxAds setting default value
	if nil == pod.MaxAds {
		pod.MaxAds = getIntPtr(3)
	}

	//pod.AdvertiserExclusionPercent setting default value
	if nil == pod.AdvertiserExclusionPercent {
		pod.AdvertiserExclusionPercent = getIntPtr(100)
	}

	//pod.IABCategoryExclusionPercent setting default value
	if nil == pod.IABCategoryExclusionPercent {
		pod.IABCategoryExclusionPercent = getIntPtr(100)
	}
}

//SetDefaultValue will set default values if not present
func (ext *ExtRequestAdPod) SetDefaultValue() {
	//ext.VideoAdPod setting default value
	ext.VideoAdPod.SetDefaultValue()

	//ext.CrossPodAdvertiserExclusionPercent setting default value
	if nil == ext.CrossPodAdvertiserExclusionPercent {
		ext.CrossPodAdvertiserExclusionPercent = getIntPtr(100)
	}

	//ext.CrossPodIABCategoryExclusionPercent setting default value
	if nil == ext.CrossPodIABCategoryExclusionPercent {
		ext.CrossPodIABCategoryExclusionPercent = getIntPtr(100)
	}

	//ext.IABCategoryExclusionWindow setting default value
	if nil == ext.IABCategoryExclusionWindow {
		ext.IABCategoryExclusionWindow = getIntPtr(0)
	}

	//ext.AdvertiserExclusionWindow setting default value
	if nil == ext.AdvertiserExclusionWindow {
		ext.AdvertiserExclusionWindow = getIntPtr(0)
	}
}

//SetDefaultValue will set default values if not present
func (ext *ExtVideoAdPod) SetDefaultValue() {
	//ext.Offset setting default values
	if nil == ext.Offset {
		ext.Offset = getIntPtr(0)
	}

	//ext.AdPod setting default values
	if nil == ext.AdPod {
		ext.AdPod = &VideoAdPod{}
	}
	ext.AdPod.SetDefaultValue()
}

//SetDefaultAdDuration will set default pod ad slot durations
func (pod *VideoAdPod) SetDefaultAdDurations(podMinDuration, podMaxDuration int64) {
	//pod.MinDuration setting default adminduration
	if nil == pod.MinDuration {
		duration := int(podMinDuration / 2)
		pod.MinDuration = &duration
	}

	//pod.MaxDuration setting default admaxduration
	if nil == pod.MaxDuration {
		duration := int(podMaxDuration / 2)
		pod.MaxDuration = &duration
	}
}

//Merge VideoAdPod Values
func (pod *VideoAdPod) Merge(parent *VideoAdPod) {
	//pod.MinAds setting default value
	if nil == pod.MinAds {
		pod.MinAds = parent.MinAds
	}

	//pod.MaxAds setting default value
	if nil == pod.MaxAds {
		pod.MaxAds = parent.MaxAds
	}

	//pod.AdvertiserExclusionPercent setting default value
	if nil == pod.AdvertiserExclusionPercent {
		pod.AdvertiserExclusionPercent = parent.AdvertiserExclusionPercent
	}

	//pod.IABCategoryExclusionPercent setting default value
	if nil == pod.IABCategoryExclusionPercent {
		pod.IABCategoryExclusionPercent = parent.IABCategoryExclusionPercent
	}
}

//ValidateAdPodDurations will validate adpod min,max durations
func (pod *VideoAdPod) ValidateAdPodDurations(minDuration, maxDuration, maxExtended int64) (err []error) {
	if minDuration < 0 {
		err = append(err, errInvalidAdPodMinDuration)
	}

	if maxDuration <= 0 {
		err = append(err, errInvalidAdPodMaxDuration)
	}

	if minDuration > maxDuration {
		err = append(err, errInvalidAdPodDuration)
	}

	//adpod.adminduration*adpod.minads should be greater than or equal to video.minduration
	if nil != pod.MinAds && nil != pod.MinDuration {
		if int64((*pod.MinAds)*(*pod.MinDuration)) < minDuration {
			err = append(err, errInvalidMinDurationRange)
		}
	}

	//adpod.admaxduration*adpod.maxads should be less than or equal to video.maxduration + video.maxextended
	if maxExtended > 0 && nil != pod.MaxAds && nil != pod.MaxDuration {
		if int64((*pod.MaxAds)*(*pod.MaxDuration)) > (maxDuration + maxExtended) {
			err = append(err, errInvalidMaxDurationRange)
		}
	}
	return
}