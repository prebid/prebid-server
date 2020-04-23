package openrtb

// 5.19 IQG Media Ratings
//
// Media ratings used in describing content based on the IQG 2.1 categorization.
// Refer to www.iab.com/guidelines/digital-video-suite for more information.
type IQGMediaRating int8

const (
	IQGMediaRatingAll    IQGMediaRating = 1 // All Audiences
	IQGMediaRatingOver12 IQGMediaRating = 2 // Everyone Over 12
	IQGMediaRatingMature IQGMediaRating = 3 // Mature Audiences
)
