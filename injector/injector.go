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
	Errors                  []string
	Impressions             []string
	VideoClicks             []string
	NonLinearClickTracking  []string
	CompanionClickThrough   []string
	LinearTrackingEvents    map[string][]string
	NonLinearTrackingEvents map[string][]string
	CompanionTrackingEvents map[string][]string
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

func (ti *TrackerInjector) InjectTracker(vastXML string, NURL string) string {
	if vastXML == "" && NURL == "" {
		// TODO Log a adapter.<bidder-name>.requests.badserverresponse
		return vastXML
	}

	if vastXML == "" {
		return fmt.Sprintf(emptyAdmResponse, NURL)
	}

	var outputXML strings.Builder
	encoder := xml.NewEncoder(&outputXML)
	st := &InjectionState{
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
		t, err := decoder.RawToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return ""
		}

		switch tt := t.(type) {
		case xml.StartElement:
			ti.handleStartElement(tt, st, &outputXML, encoder)
		case xml.EndElement:
			ti.handleEndElement(tt, st, &outputXML, encoder)
		case xml.CharData:
			tt2 := strings.Trim(string(tt), trimRunes)
			if len(tt2) != 0 {
				encoder.Flush()
				outputXML.WriteString("<![CDATA[")
				outputXML.WriteString(tt2)
				outputXML.WriteString("]]>")
				continue
			}
		default:
			encoder.EncodeToken(t)
		}
	}

	encoder.Flush()

	if !st.inlineWrapperTagFound {
		// 	// Todo log adapter.<bidder-name>.requests.badserverresponse metrics
		return vastXML
	}
	return outputXML.String()
}

func (ti *TrackerInjector) handleStartElement(tt xml.StartElement, st *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) {
	switch tt.Name.Local {
	case wrapperCase:
		st.wrapperTagFound = true
		encoder.EncodeToken(tt)
	case creativeCase:
		st.isCreative = true
		for _, attr := range tt.Attr {
			if strings.ToLower(attr.Name.Local) == "adid" {
				st.creativeId = attr.Value
			}
		}
		encoder.EncodeToken(tt)
	case linearCase:
		st.injectVideoClicks = true
		st.injectTracker = true
		encoder.EncodeToken(tt)
	case videoClicksCase:
		st.injectVideoClicks = false
		encoder.Flush()
		encoder.EncodeToken(tt)
		encoder.Flush()
		ti.addClickTrackingEvent(outputXML, st.creativeId, false)
	case nonLinearAdsCase:
		st.injectTracker = true
		encoder.EncodeToken(tt)
	case trackingEventsCase:
		if st.isCreative {
			st.injectTracker = false
			encoder.Flush()
			encoder.EncodeToken(tt)
			encoder.Flush()
			ti.addTrackingEvent(outputXML, st.creativeId, false)
		}
	default:
		encoder.EncodeToken(tt)
	}
}

func (ti *TrackerInjector) handleEndElement(tt xml.EndElement, st *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) {
	switch tt.Name.Local {
	case impressionCase:
		encoder.Flush()
		encoder.EncodeToken(tt)
		encoder.Flush()
		if !st.impressionTagFound {
			ti.addImpressionTrackingEvent(outputXML)
			st.impressionTagFound = true
		}
	case errorCase:
		encoder.Flush()
		encoder.EncodeToken(tt)
		encoder.Flush()
		if !st.errorTagFound {
			ti.addErrorTrackingEvent(outputXML)
			st.errorTagFound = true
		}
	case nonLinearAdsCase:
		if st.injectTracker {
			st.injectTracker = false
			encoder.Flush()
			ti.addTrackingEvent(outputXML, st.creativeId, true)
			if !st.nonLinearTagFound && st.wrapperTagFound {
				ti.addNonLinearClickTrackingEvent(outputXML, st.creativeId, true)
			}
			encoder.EncodeToken(tt)
		}
	case linearCase:
		if st.injectVideoClicks {
			st.injectVideoClicks = false
			encoder.Flush()
			ti.addClickTrackingEvent(outputXML, st.creativeId, true)
		}
		if st.injectTracker {
			st.injectTracker = false
			encoder.Flush()
			ti.addTrackingEvent(outputXML, st.creativeId, true)
		}
		encoder.EncodeToken(tt)
	case inlineCase, wrapperCase:
		st.wrapperTagFound = false
		st.inlineWrapperTagFound = true
		encoder.Flush()
		if !st.impressionTagFound {
			ti.addImpressionTrackingEvent(outputXML)
		}
		st.impressionTagFound = false
		if !st.errorTagFound {
			ti.addErrorTrackingEvent(outputXML)
		}
		st.errorTagFound = false
		encoder.EncodeToken(tt)
	case nonLinearCase:
		encoder.Flush()
		ti.addNonLinearClickTrackingEvent(outputXML, st.creativeId, false)
		st.nonLinearTagFound = true
		encoder.EncodeToken(tt)
	case companionCase:
		st.companionTagFound = true
		encoder.Flush()
		ti.addCompanionClickThroughEvent(outputXML, st.creativeId, false)
		encoder.EncodeToken(tt)
	case creativeCase:
		st.isCreative = false
		encoder.EncodeToken(tt)
	case companionAdsCase:
		if !st.companionTagFound && st.wrapperTagFound {
			encoder.Flush()
			ti.addCompanionClickThroughEvent(outputXML, st.creativeId, true)
		}
		encoder.EncodeToken(tt)
	default:
		encoder.EncodeToken(tt)
	}
}

func (ti *TrackerInjector) addTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(trackingEventStartTag)
	}
	for typ, urls := range ti.events.LinearTrackingEvents {
		ti.writeTrackingEvent(urls, outputXML, "<Tracking event=\""+string(typ)+"\"><![CDATA[", "]]></Tracking>", creativeId, typ, tracking)
	}
	if addParentTag {
		outputXML.WriteString(trackingEventEndTag)
	}
}

func (ti *TrackerInjector) addClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(videoClickStartTag)
	}
	ti.writeTrackingEvent(ti.events.VideoClicks, outputXML, clickTrackingStartTag, clickTrackingEndTag, creativeId, "", clicktracking)
	if addParentTag {
		outputXML.WriteString(videoClickEndTag)
	}
}

func (ti *TrackerInjector) addImpressionTrackingEvent(outputXML *strings.Builder) {
	ti.writeTrackingEvent(ti.events.Impressions, outputXML, impressionStartTag, impressionEndTag, "", "", impression)
}

func (ti *TrackerInjector) addErrorTrackingEvent(outputXML *strings.Builder) {
	ti.writeTrackingEvent(ti.events.Errors, outputXML, errorStartTag, errorEndTag, "", "", err)
}

func (ti *TrackerInjector) addNonLinearClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(nonLinearStartTag)
	}
	ti.writeTrackingEvent(ti.events.NonLinearClickTracking, outputXML, nonLinearClickTrackingStartTag, nonLinearClickTrackingEndTag, creativeId, "", nonlinearclicktracking)
	if addParentTag {
		outputXML.WriteString(nonLinearEndTag)
	}
}

func (ti *TrackerInjector) addCompanionClickThroughEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString(companionStartTag)
	}
	ti.writeTrackingEvent(ti.events.CompanionClickThrough, outputXML, companionClickThroughStartTag, companionClickThroughEndTag, creativeId, "", companionclickthrough)
	if addParentTag {
		outputXML.WriteString(companionEndTag)
	}
}

func (ti *TrackerInjector) writeTrackingEvent(urls []string, outputXML *strings.Builder, startTag, endTag, creativeId, eventType, vastEvent string) {
	ti.provider.PopulateEventMacros(creativeId, eventType, vastEvent)
	for _, url := range urls {
		outputXML.WriteString(startTag)
		ti.replacer.Replace(outputXML, url, ti.provider)
		outputXML.WriteString(endTag)
	}
}
