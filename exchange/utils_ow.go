package exchange

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func JLogf(msg string, obj interface{}) {
	if glog.V(3) {
		data, _ := json.Marshal(obj)
		glog.Infof("[OPENWRAP] %v:%v", msg, string(data))
	}
}

// updateContentObjectForBidder updates the content object for each bidder based on content transparency rules
func updateContentObjectForBidder(allBidderRequests []BidderRequest, requestExt *openrtb_ext.ExtRequest) {
	if requestExt == nil || requestExt.Prebid.Transparency == nil || requestExt.Prebid.Transparency.Content == nil {
		return
	}

	rules := requestExt.Prebid.Transparency.Content

	if len(rules) == 0 {
		return
	}

	var contentObject *openrtb2.Content
	isApp := false
	bidderRequest := allBidderRequests[0]
	if bidderRequest.BidRequest.App != nil && bidderRequest.BidRequest.App.Content != nil {
		contentObject = bidderRequest.BidRequest.App.Content
		isApp = true
	} else if bidderRequest.BidRequest.Site != nil && bidderRequest.BidRequest.Site.Content != nil {
		contentObject = bidderRequest.BidRequest.Site.Content
	} else {
		return
	}

	// Dont send content object if no rule and default is not present
	var defaultRule = openrtb_ext.TransparencyRule{}
	if rule, ok := rules["default"]; ok {
		defaultRule = rule
	}

	for _, bidderRequest := range allBidderRequests {
		var newContentObject *openrtb2.Content

		rule, ok := rules[string(bidderRequest.BidderName)]
		if !ok {
			rule = defaultRule
		}

		if len(rule.Keys) != 0 {
			newContentObject = createNewContentObject(contentObject, rule.Include, rule.Keys)
		} else if rule.Include {
			newContentObject = contentObject
		}
		deepCopyContentObj(bidderRequest.BidRequest, newContentObject, isApp)
	}
}

func deepCopyContentObj(request *openrtb2.BidRequest, contentObject *openrtb2.Content, isApp bool) {
	if isApp {
		app := *request.App
		app.Content = contentObject
		request.App = &app
	} else {
		site := *request.Site
		site.Content = contentObject
		request.Site = &site
	}
}

// func createNewContentObject(contentObject *openrtb2.Content, include bool, keys []string) *openrtb2.Content {
// 	if include {
// 		return includeKeys(contentObject, keys)
// 	}
// 	return excludeKeys(contentObject, keys)

// }

// func excludeKeys(contentObject *openrtb2.Content, keys []string) *openrtb2.Content {
// 	newContentObject := *contentObject

// 	keyMap := make(map[string]struct{}, 1)
// 	for _, key := range keys {
// 		keyMap[key] = struct{}{}
// 	}

// 	rt := reflect.TypeOf(newContentObject)
// 	for i := 0; i < rt.NumField(); i++ {
// 		key := strings.Split(rt.Field(i).Tag.Get("json"), ",")[0] // remove omitempty, etc
// 		if _, ok := keyMap[key]; ok {
// 			reflect.ValueOf(&newContentObject).Elem().FieldByName(rt.Field(i).Name).Set(reflect.Zero(rt.Field(i).Type))
// 		}
// 	}

// 	return &newContentObject
// }

// func includeKeys(contentObject *openrtb2.Content, keys []string) *openrtb2.Content {
// 	newContentObject := openrtb2.Content{}
// 	v := reflect.ValueOf(contentObject).Elem()
// 	keyMap := make(map[string]struct{}, 1)
// 	for _, key := range keys {
// 		keyMap[key] = struct{}{}
// 	}

// 	rt := reflect.TypeOf(newContentObject)
// 	rvElem := reflect.ValueOf(&newContentObject).Elem()
// 	for i := 0; i < rt.NumField(); i++ {
// 		field := rt.Field(i)
// 		key := strings.Split(field.Tag.Get("json"), ",")[0] // remove omitempty, etc
// 		if _, ok := keyMap[key]; ok {
// 			rvElem.FieldByName(field.Name).Set(v.FieldByName(field.Name))
// 		}
// 	}

// 	return &newContentObject
// }

func createNewContentObject(contentObject *openrtb2.Content, include bool, keys []string) *openrtb2.Content {
	newContentObject := &openrtb2.Content{}
	if !include {
		*newContentObject = *contentObject
		for _, key := range keys {

			switch key {
			case "id":
				newContentObject.ID = ""
			case "episode":
				newContentObject.Episode = 0
			case "title":
				newContentObject.Title = ""
			case "series":
				newContentObject.Series = ""
			case "season":
				newContentObject.Season = ""
			case "artist":
				newContentObject.Artist = ""
			case "genre":
				newContentObject.Genre = ""
			case "album":
				newContentObject.Album = ""
			case "isrc":
				newContentObject.ISRC = ""
			case "producer":
				newContentObject.Producer = nil
			case "url":
				newContentObject.URL = ""
			case "cat":
				newContentObject.Cat = nil
			case "prodq":
				newContentObject.ProdQ = nil
			case "videoquality":
				newContentObject.VideoQuality = nil
			case "context":
				newContentObject.Context = 0
			case "contentrating":
				newContentObject.ContentRating = ""
			case "userrating":
				newContentObject.UserRating = ""
			case "qagmediarating":
				newContentObject.QAGMediaRating = 0
			case "keywords":
				newContentObject.Keywords = ""
			case "livestream":
				newContentObject.LiveStream = 0
			case "sourcerelationship":
				newContentObject.SourceRelationship = 0
			case "len":
				newContentObject.Len = 0
			case "language":
				newContentObject.Language = ""
			case "embeddable":
				newContentObject.Embeddable = 0
			case "data":
				newContentObject.Data = nil
			case "ext":
				newContentObject.Ext = nil
			}

		}
		return newContentObject
	}

	for _, key := range keys {
		switch key {
		case "id":
			newContentObject.ID = contentObject.ID
		case "episode":
			newContentObject.Episode = contentObject.Episode
		case "title":
			newContentObject.Title = contentObject.Title
		case "series":
			newContentObject.Series = contentObject.Series
		case "season":
			newContentObject.Season = contentObject.Season
		case "artist":
			newContentObject.Artist = contentObject.Artist
		case "genre":
			newContentObject.Genre = contentObject.Genre
		case "album":
			newContentObject.Album = contentObject.Album
		case "isrc":
			newContentObject.ISRC = contentObject.ISRC
		case "producer":
			if contentObject.Producer != nil {
				producer := *contentObject.Producer
				newContentObject.Producer = &producer
			}
		case "url":
			newContentObject.URL = contentObject.URL
		case "cat":
			newContentObject.Cat = contentObject.Cat
		case "prodq":
			if contentObject.ProdQ != nil {
				prodQ := *contentObject.ProdQ
				newContentObject.ProdQ = &prodQ
			}
		case "videoquality":
			if contentObject.VideoQuality != nil {
				videoQuality := *contentObject.VideoQuality
				newContentObject.VideoQuality = &videoQuality
			}
		case "context":
			newContentObject.Context = contentObject.Context
		case "contentrating":
			newContentObject.ContentRating = contentObject.ContentRating
		case "userrating":
			newContentObject.UserRating = contentObject.UserRating
		case "qagmediarating":
			newContentObject.QAGMediaRating = contentObject.QAGMediaRating
		case "keywords":
			newContentObject.Keywords = contentObject.Keywords
		case "livestream":
			newContentObject.LiveStream = contentObject.LiveStream
		case "sourcerelationship":
			newContentObject.SourceRelationship = contentObject.SourceRelationship
		case "len":
			newContentObject.Len = contentObject.Len
		case "language":
			newContentObject.Language = contentObject.Language
		case "embeddable":
			newContentObject.Embeddable = contentObject.Embeddable
		case "data":
			if contentObject.Data != nil {
				newContentObject.Data = contentObject.Data
			}
		case "ext":
			newContentObject.Ext = contentObject.Ext
		}
	}

	return newContentObject
}
