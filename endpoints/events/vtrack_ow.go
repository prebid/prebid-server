package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/beevik/etree"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// standard VAST macros
// https://interactiveadvertisingbureau.github.io/vast/vast4macros/vast4-macros-latest.html#macro-spec-adcount
const (
	VASTAdTypeMacro    = "[ADTYPE]"
	VASTAppBundleMacro = "[APPBUNDLE]"
	VASTDomainMacro    = "[DOMAIN]"
	VASTPageURLMacro   = "[PAGEURL]"

	// PBS specific macros
	PBSEventIDMacro = "[EVENT_ID]" // macro for injecting PBS defined  video event tracker id
	//[PBS-ACCOUNT] represents publisher id / account id
	PBSAccountMacro = "[PBS-ACCOUNT]"
	// [PBS-BIDDER] represents bidder name
	PBSBidderMacro = "[PBS-BIDDER]"
	// [PBS-ORIG_BIDID] represents original bid id.
	PBSOrigBidIDMacro = "[PBS-ORIG_BIDID]"
	// [PBS-BIDID] represents bid id. If auction.generate-bid-id config is on, then resolve with response.seatbid.bid.ext.prebid.bidid. Else replace with response.seatbid.bid.id
	PBSBidIDMacro = "[PBS-BIDID]"
	// [ADERVERTISER_NAME] represents advertiser name
	PBSAdvertiserNameMacro = "[ADVERTISER_NAME]"
	// Pass imp.tagId using this macro
	PBSAdUnitIDMacro = "[AD_UNIT]"
	//PBSBidderCodeMacro represents an alias id or core bidder id.
	PBSBidderCodeMacro = "[BIDDER_CODE]"
)

var trackingEvents = []string{"start", "firstQuartile", "midpoint", "thirdQuartile", "complete"}

// PubMatic specific event IDs
// This will go in event-config once PreBid modular design is in place
var eventIDMap = map[string]string{
	"start":         "2",
	"firstQuartile": "4",
	"midpoint":      "3",
	"thirdQuartile": "5",
	"complete":      "6",
}

//InjectVideoEventTrackers injects the video tracking events
//Returns VAST xml contains as first argument. Second argument indicates whether the trackers are injected and last argument indicates if there is any error in injecting the trackers
func InjectVideoEventTrackers(trackerURL, vastXML string, bid *openrtb2.Bid, prebidGenBidId, requestingBidder, bidderCoreName, accountID string, timestamp int64, bidRequest *openrtb2.BidRequest) ([]byte, bool, error) {
	// parse VAST
	doc := etree.NewDocument()
	err := doc.ReadFromString(vastXML)
	if nil != err {
		err = fmt.Errorf("Error parsing VAST XML. '%v'", err.Error())
		glog.Errorf(err.Error())
		return []byte(vastXML), false, err // false indicates events trackers are not injected
	}

	//Maintaining BidRequest Impression Map (Copied from exchange.go#applyCategoryMapping)
	//TODO: It should be optimized by forming once and reusing
	impMap := make(map[string]*openrtb2.Imp)
	for i := range bidRequest.Imp {
		impMap[bidRequest.Imp[i].ID] = &bidRequest.Imp[i]
	}

	eventURLMap := GetVideoEventTracking(trackerURL, bid, prebidGenBidId, requestingBidder, bidderCoreName, accountID, timestamp, bidRequest, doc, impMap)
	trackersInjected := false
	// return if if no tracking URL
	if len(eventURLMap) == 0 {
		return []byte(vastXML), false, errors.New("Event URLs are not found")
	}

	creatives := FindCreatives(doc)

	if adm := strings.TrimSpace(bid.AdM); adm == "" || strings.HasPrefix(adm, "http") {
		// determine which creative type to be created based on linearity
		if imp, ok := impMap[bid.ImpID]; ok && nil != imp.Video {
			// create creative object
			creatives = doc.FindElements("VAST/Ad/Wrapper/Creatives")
			// var creative *etree.Element
			// if len(creatives) > 0 {
			// 	creative = creatives[0] // consider only first creative
			// } else {
			creative := doc.CreateElement("Creative")
			creatives[0].AddChild(creative)

			// }

			switch imp.Video.Linearity {
			case openrtb2.VideoLinearityLinearInStream:
				creative.AddChild(doc.CreateElement("Linear"))
			case openrtb2.VideoLinearityNonLinearOverlay:
				creative.AddChild(doc.CreateElement("NonLinearAds"))
			default: // create both type of creatives
				creative.AddChild(doc.CreateElement("Linear"))
				creative.AddChild(doc.CreateElement("NonLinearAds"))
			}
			creatives = creative.ChildElements() // point to actual cratives
		}
	}
	for _, creative := range creatives {
		trackingEvents := creative.SelectElement("TrackingEvents")
		if nil == trackingEvents {
			trackingEvents = creative.CreateElement("TrackingEvents")
			creative.AddChild(trackingEvents)
		}
		// Inject
		for event, url := range eventURLMap {
			trackingEle := trackingEvents.CreateElement("Tracking")
			trackingEle.CreateAttr("event", event)
			trackingEle.SetText(fmt.Sprintf("%s", url))
			trackersInjected = true
		}
	}

	out := []byte(vastXML)
	var wErr error
	if trackersInjected {
		out, wErr = doc.WriteToBytes()
		trackersInjected = trackersInjected && nil == wErr
		if nil != wErr {
			glog.Errorf("%v", wErr.Error())
		}
	}
	return out, trackersInjected, wErr
}

// GetVideoEventTracking returns map containing key as event name value as associaed video event tracking URL
// By default PBS will expect [EVENT_ID] macro in trackerURL to inject event information
// [EVENT_ID] will be injected with one of the following values
//    firstQuartile, midpoint, thirdQuartile, complete
// If your company can not use [EVENT_ID] and has its own macro. provide config.TrackerMacros implementation
// and ensure that your macro is part of trackerURL configuration
func GetVideoEventTracking(trackerURL string, bid *openrtb2.Bid, prebidGenBidId, requestingBidder string, bidderCoreName string, accountId string, timestamp int64, req *openrtb2.BidRequest, doc *etree.Document, impMap map[string]*openrtb2.Imp) map[string]string {
	eventURLMap := make(map[string]string)
	if "" == strings.TrimSpace(trackerURL) {
		return eventURLMap
	}

	// lookup custom macros
	var customMacroMap map[string]string
	if nil != req.Ext {
		reqExt := new(openrtb_ext.ExtRequest)
		err := json.Unmarshal(req.Ext, &reqExt)
		if err == nil {
			customMacroMap = reqExt.Prebid.Macros
		} else {
			glog.Warningf("Error in unmarshling req.Ext.Prebid.Vast: [%s]", err.Error())
		}
	}

	for _, event := range trackingEvents {
		eventURL := trackerURL
		// lookup in custom macros
		if nil != customMacroMap {
			for customMacro, value := range customMacroMap {
				eventURL = replaceMacro(eventURL, customMacro, value)
			}
		}
		// replace standard macros
		eventURL = replaceMacro(eventURL, VASTAdTypeMacro, string(openrtb_ext.BidTypeVideo))
		if nil != req && nil != req.App {
			// eventURL = replaceMacro(eventURL, VASTAppBundleMacro, req.App.Bundle)
			eventURL = replaceMacro(eventURL, VASTDomainMacro, req.App.Bundle)
			if nil != req.App.Publisher {
				eventURL = replaceMacro(eventURL, PBSAccountMacro, req.App.Publisher.ID)
			}
		}
		if nil != req && nil != req.Site {
			eventURL = replaceMacro(eventURL, VASTDomainMacro, getDomain(req.Site))
			eventURL = replaceMacro(eventURL, VASTPageURLMacro, req.Site.Page)
			if nil != req.Site.Publisher {
				eventURL = replaceMacro(eventURL, PBSAccountMacro, req.Site.Publisher.ID)
			}
		}

		domain := ""
		if len(bid.ADomain) > 0 {
			var err error
			//eventURL = replaceMacro(eventURL, PBSAdvertiserNameMacro, strings.Join(bid.ADomain, ","))
			domain, err = extractDomain(bid.ADomain[0])
			if err != nil {
				glog.Warningf("Unable to extract domain from '%s'. [%s]", bid.ADomain[0], err.Error())
			}
		}

		eventURL = replaceMacro(eventURL, PBSAdvertiserNameMacro, domain)

		eventURL = replaceMacro(eventURL, PBSBidderMacro, bidderCoreName)
		eventURL = replaceMacro(eventURL, PBSBidderCodeMacro, requestingBidder)

		/* Use generated bidId if present, else use bid.ID */
		if len(prebidGenBidId) > 0 && prebidGenBidId != bid.ID {
			eventURL = replaceMacro(eventURL, PBSBidIDMacro, prebidGenBidId)
		} else {
			eventURL = replaceMacro(eventURL, PBSBidIDMacro, bid.ID)
		}
		eventURL = replaceMacro(eventURL, PBSOrigBidIDMacro, bid.ID)

		// replace [EVENT_ID] macro with PBS defined event ID
		eventURL = replaceMacro(eventURL, PBSEventIDMacro, eventIDMap[event])

		if imp, ok := impMap[bid.ImpID]; ok {
			eventURL = replaceMacro(eventURL, PBSAdUnitIDMacro, imp.TagID)
		} else {
			glog.Warningf("Setting empty value for %s macro, as failed to determine imp.TagID for bid.ImpID: %s", PBSAdUnitIDMacro, bid.ImpID)
			eventURL = replaceMacro(eventURL, PBSAdUnitIDMacro, "")
		}

		eventURLMap[event] = eventURL
	}
	return eventURLMap
}

func replaceMacro(trackerURL, macro, value string) string {
	macro = strings.TrimSpace(macro)
	trimmedValue := strings.TrimSpace(value)

	if strings.HasPrefix(macro, "[") && strings.HasSuffix(macro, "]") && len(trimmedValue) > 0 {
		trackerURL = strings.ReplaceAll(trackerURL, macro, url.QueryEscape(value))
	} else if strings.HasPrefix(macro, "[") && strings.HasSuffix(macro, "]") && len(trimmedValue) == 0 {
		trackerURL = strings.ReplaceAll(trackerURL, macro, url.QueryEscape(""))
	} else {
		glog.Warningf("Invalid macro '%v'. Either empty or missing prefix '[' or suffix ']", macro)
	}
	return trackerURL
}

//FindCreatives finds Linear, NonLinearAds fro InLine and Wrapper Type of creatives
//from input doc - VAST Document
//NOTE: This function is temporarily seperated to reuse in ctv_auction.go. Because, in case of ctv
//we generate bid.id
func FindCreatives(doc *etree.Document) []*etree.Element {
	// Find Creatives of Linear and NonLinear Type
	// Injecting Tracking Events for Companion is not supported here
	creatives := doc.FindElements("VAST/Ad/InLine/Creatives/Creative/Linear")
	creatives = append(creatives, doc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/Linear")...)
	creatives = append(creatives, doc.FindElements("VAST/Ad/InLine/Creatives/Creative/NonLinearAds")...)
	creatives = append(creatives, doc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/NonLinearAds")...)
	return creatives
}

func extractDomain(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "http://" + rawURL
	}
	// decode rawURL
	rawURL, err := url.QueryUnescape(rawURL)
	if nil != err {
		return "", err
	}
	url, err := url.Parse(rawURL)
	if nil != err {
		return "", err
	}
	// remove www if present
	return strings.TrimPrefix(url.Hostname(), "www."), nil
}

func getDomain(site *openrtb2.Site) string {
	if site.Domain != "" {
		return site.Domain
	}

	hostname := ""

	if site.Page != "" {
		pageURL, err := url.Parse(site.Page)
		if err == nil && pageURL != nil {
			hostname = pageURL.Host
		}
	}
	return hostname
}
