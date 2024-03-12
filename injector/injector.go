package injector

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/metrics"
)

const (
	emptyAdmResponse = `<VAST version="3.0"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[%s]]></VASTAdTagURI><Creatives></Creatives></Wrapper></Ad></VAST>`
)

const (
	companionStartTag              = "<Companion>"
	companionEndTag                = "</Companion>"
	nonLinearStartTag              = "<NonLinear>"
	nonLinearEndTag                = "</NonLinear>"
	videoClickStartTag             = "<VideoClicks>"
	videoClickEndTag               = "</VideoClicks>"
	trackingEventStartTag          = "<TrackingEvents>"
	trackingEventEndTag            = "</TrackingEvents>"
	clickTrackingStartTag          = "<ClickTracking><![CDATA["
	clickTrackingEndTag            = "]]></ClickTracking>"
	impressionStartTag             = "<Impression><![CDATA["
	impressionEndTag               = "]]></Impression>"
	errorStartTag                  = "<Error><![CDATA["
	errorEndTag                    = "]]></Error>"
	nonLinearClickTrackingStartTag = "<NonLinearClickTracking><![CDATA["
	nonLinearClickTrackingEndTag   = "]]></NonLinearClickTracking>"
	companionClickThroughStartTag  = "<CompanionClickThrough><![CDATA["
	companionClickThroughEndTag    = "]]></CompanionClickThrough>"
	tracking                       = "tracking"
	companionclickthrough          = "companionclickthrough"
	nonlinearclicktracking         = "nonlinearclicktracking"
	impression                     = "impression"
	err                            = "error"
	clicktracking                  = "clicktracking"
	adId                           = "adid"
	trackingStartTag               = `<Tracking event="%s"><![CDATA[`
	trackingEndTag                 = "]]></Tracking>"
)

const (
	inlineCase         = "InLine"
	wrapperCase        = "Wrapper"
	creativeCase       = "Creative"
	linearCase         = "Linear"
	nonLinearCase      = "NonLinear"
	videoClicksCase    = "VideoClicks"
	nonLinearAdsCase   = "NonLinearAds"
	trackingEventsCase = "TrackingEvents"
	impressionCase     = "Impression"
	errorCase          = "Error"
	companionCase      = "Companion"
	companionAdsCase   = "CompanionAds"
)

type Injector interface {
	InjectTracker(vastXML string, NURL string) string
}

type VASTEvents struct {
	Errors                 []string
	Impressions            []string
	VideoClicks            []string
	NonLinearClickTracking []string
	CompanionClickThrough  []string
	TrackingEvents         map[string][]string
}

type InjectionState struct {
	injectTracker         bool
	injectVideoClicks     bool
	inlineWrapperTagFound bool
	wrapperTagFound       bool
	impressionTagFound    bool
	errorTagFound         bool
	creativeId            string
	isCreative            bool
	companionTagFound     bool
	nonLinearTagFound     bool
}

type TrackerInjector struct {
	replacer macros.Replacer
	events   VASTEvents
	me       metrics.MetricsEngine
	provider *macros.MacroProvider
}

var trimRunes = "\t\r\b\n "

func NewTrackerInjector(replacer macros.Replacer, provider *macros.MacroProvider, events VASTEvents) *TrackerInjector {
	return &TrackerInjector{
		replacer: replacer,
		provider: provider,
		events:   events,
	}
}

func (trackerinjector *TrackerInjector) InjectTracker(vastXML string, NURL string) string {
	if vastXML == "" && NURL == "" {
		// TODO Log a adapter.<bidder-name>.requests.badserverresponse
		return vastXML
	}

	if vastXML == "" {
		return fmt.Sprintf(emptyAdmResponse, NURL)
	}

	var outputXML strings.Builder
	encoder := xml.NewEncoder(&outputXML)
	state := &InjectionState{
		injectTracker:         false,
		injectVideoClicks:     false,
		inlineWrapperTagFound: false,
		wrapperTagFound:       false,
		impressionTagFound:    false,
		errorTagFound:         false,
		creativeId:            "",
		isCreative:            false,
		companionTagFound:     false,
		nonLinearTagFound:     false,
	}

	reader := strings.NewReader(vastXML)
	decoder := xml.NewDecoder(reader)

	for {
		rawToken, err := decoder.RawToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return ""
		}

		switch token := rawToken.(type) {
		case xml.StartElement:
			trackerinjector.handleStartElement(token, state, &outputXML, encoder)
		case xml.EndElement:
			trackerinjector.handleEndElement(token, state, &outputXML, encoder)
		case xml.CharData:
			charData := strings.Trim(string(token), trimRunes)
			if len(charData) != 0 {
				encoder.Flush()
				outputXML.WriteString("<![CDATA[")
				outputXML.WriteString(charData)
				outputXML.WriteString("]]>")
				continue
			}
		default:
			encoder.EncodeToken(rawToken)
		}
	}

	encoder.Flush()

	if !state.inlineWrapperTagFound {
		// Todo log adapter.<bidder-name>.requests.badserverresponse metrics
		return vastXML
	}
	return outputXML.String()
}

func (trackerinjector *TrackerInjector) handleStartElement(token xml.StartElement, state *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) {
	switch token.Name.Local {
	case wrapperCase:
		state.wrapperTagFound = true
		encoder.EncodeToken(token)
	case creativeCase:
		state.isCreative = true
		for _, attr := range token.Attr {
			if strings.ToLower(attr.Name.Local) == adId {
				state.creativeId = attr.Value
			}
		}
		encoder.EncodeToken(token)
	case linearCase:
		state.injectVideoClicks = true
		state.injectTracker = true
		encoder.EncodeToken(token)
	case videoClicksCase:
		state.injectVideoClicks = false
		encoder.EncodeToken(token)
		encoder.Flush()
		trackerinjector.addClickTrackingEvent(outputXML, state.creativeId, false)
	case nonLinearAdsCase:
		state.injectTracker = true
		encoder.EncodeToken(token)
	case trackingEventsCase:
		if state.isCreative {
			state.injectTracker = false
			encoder.EncodeToken(token)
			encoder.Flush()
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, false)
		}
	default:
		encoder.EncodeToken(token)
	}
}

func (trackerinjector *TrackerInjector) handleEndElement(token xml.EndElement, state *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) {
	switch token.Name.Local {
	case impressionCase:
		fmt.Println(outputXML.String())
		encoder.EncodeToken(token)
		encoder.Flush()
		if !state.impressionTagFound {
			trackerinjector.addImpressionTrackingEvent(outputXML)
			state.impressionTagFound = true
		}
	case errorCase:
		encoder.EncodeToken(token)
		encoder.Flush()
		if !state.errorTagFound {
			trackerinjector.addErrorTrackingEvent(outputXML)
			state.errorTagFound = true
		}
	case nonLinearAdsCase:
		if state.injectTracker {
			state.injectTracker = false
			encoder.Flush()
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, true)
			if !state.nonLinearTagFound && state.wrapperTagFound {
				trackerinjector.addNonLinearClickTrackingEvent(outputXML, state.creativeId, true)
			}
			encoder.EncodeToken(token)
		}
	case linearCase:
		if state.injectVideoClicks {
			state.injectVideoClicks = false
			encoder.Flush()
			trackerinjector.addClickTrackingEvent(outputXML, state.creativeId, true)
		}
		if state.injectTracker {
			state.injectTracker = false
			encoder.Flush()
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, true)
		}
		encoder.EncodeToken(token)
	case inlineCase, wrapperCase:
		state.wrapperTagFound = false
		state.inlineWrapperTagFound = true
		encoder.Flush()
		if !state.impressionTagFound {
			trackerinjector.addImpressionTrackingEvent(outputXML)
		}
		state.impressionTagFound = false
		if !state.errorTagFound {
			trackerinjector.addErrorTrackingEvent(outputXML)
		}
		state.errorTagFound = false
		encoder.EncodeToken(token)
	case nonLinearCase:
		encoder.Flush()
		trackerinjector.addNonLinearClickTrackingEvent(outputXML, state.creativeId, false)
		state.nonLinearTagFound = true
		encoder.EncodeToken(token)
	case companionCase:
		state.companionTagFound = true
		encoder.Flush()
		trackerinjector.addCompanionClickThroughEvent(outputXML, state.creativeId, false)
		encoder.EncodeToken(token)
	case creativeCase:
		state.isCreative = false
		encoder.EncodeToken(token)
	case companionAdsCase:
		if !state.companionTagFound && state.wrapperTagFound {
			encoder.Flush()
			trackerinjector.addCompanionClickThroughEvent(outputXML, state.creativeId, true)
		}
		encoder.EncodeToken(token)
	default:
		encoder.EncodeToken(token)
	}
}

func (trackerinjector *TrackerInjector) addTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(trackingEventStartTag)
	}
	for typ, urls := range trackerinjector.events.TrackingEvents {
		trackerinjector.writeTrackingEvent(urls, outputXML, fmt.Sprintf(trackingStartTag, typ), trackingEndTag, creativeId, typ, tracking)
	}
	if addParentTag {
		outputXML.WriteString(trackingEventEndTag)
	}
}

func (trackerinjector *TrackerInjector) addClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(videoClickStartTag)
	}
	trackerinjector.writeTrackingEvent(trackerinjector.events.VideoClicks, outputXML, clickTrackingStartTag, clickTrackingEndTag, creativeId, "", clicktracking)
	if addParentTag {
		outputXML.WriteString(videoClickEndTag)
	}
}

func (trackerinjector *TrackerInjector) addImpressionTrackingEvent(outputXML *strings.Builder) {
	trackerinjector.writeTrackingEvent(trackerinjector.events.Impressions, outputXML, impressionStartTag, impressionEndTag, "", "", impression)
}

func (trackerinjector *TrackerInjector) addErrorTrackingEvent(outputXML *strings.Builder) {
	trackerinjector.writeTrackingEvent(trackerinjector.events.Errors, outputXML, errorStartTag, errorEndTag, "", "", err)
}

func (trackerinjector *TrackerInjector) addNonLinearClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(nonLinearStartTag)
	}
	trackerinjector.writeTrackingEvent(trackerinjector.events.NonLinearClickTracking, outputXML, nonLinearClickTrackingStartTag, nonLinearClickTrackingEndTag, creativeId, "", nonlinearclicktracking)
	if addParentTag {
		outputXML.WriteString(nonLinearEndTag)
	}
}

func (trackerinjector *TrackerInjector) addCompanionClickThroughEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(companionStartTag)
	}
	trackerinjector.writeTrackingEvent(trackerinjector.events.CompanionClickThrough, outputXML, companionClickThroughStartTag, companionClickThroughEndTag, creativeId, "", companionclickthrough)
	if addParentTag {
		outputXML.WriteString(companionEndTag)
	}
}

func (trackerinjector *TrackerInjector) writeTrackingEvent(urls []string, outputXML *strings.Builder, startTag, endTag, creativeId, eventType, vastEvent string) {
	trackerinjector.provider.PopulateEventMacros(creativeId, eventType, vastEvent)
	for _, url := range urls {
		outputXML.WriteString(startTag)
		trackerinjector.replacer.Replace(outputXML, url, trackerinjector.provider)
		outputXML.WriteString(endTag)
	}
}
