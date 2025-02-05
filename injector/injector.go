package injector

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
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

func (trackerinjector *TrackerInjector) InjectTracker(vastXML string, NURL string) (string, error) {
	if vastXML == "" && NURL == "" {
		// TODO Log a adapter.<bidder-name>.requests.badserverresponse
		return vastXML, fmt.Errorf("invalid Vast XML")
	}

	if vastXML == "" {
		return fmt.Sprintf(emptyAdmResponse, NURL), nil
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
			} else {
				return "", fmt.Errorf("XML processing error: %w", err)
			}
		}

		switch token := rawToken.(type) {
		case xml.StartElement:
			err = trackerinjector.handleStartElement(token, state, &outputXML, encoder)
		case xml.EndElement:
			err = trackerinjector.handleEndElement(token, state, &outputXML, encoder)
		case xml.CharData:
			charData := strings.Trim(string(token), trimRunes)
			if len(charData) != 0 {
				err = encoder.Flush()
				outputXML.WriteString("<![CDATA[" + charData + "]]>")
			}
		default:
			err = encoder.EncodeToken(rawToken)
		}

		if err != nil {
			return "", fmt.Errorf("XML processing error: %w", err)
		}
	}

	if err := encoder.Flush(); err != nil {
		return "", fmt.Errorf("XML processing error: %w", err)
	}

	if !state.inlineWrapperTagFound {
		// Todo log adapter.<bidder-name>.requests.badserverresponse metrics
		return vastXML, fmt.Errorf("invalid VastXML, inline/wrapper tag not found")
	}
	return outputXML.String(), nil
}

func (trackerinjector *TrackerInjector) handleStartElement(token xml.StartElement, state *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) error {
	var err error
	switch token.Name.Local {
	case wrapperCase:
		state.wrapperTagFound = true
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case creativeCase:
		state.isCreative = true
		for _, attr := range token.Attr {
			if strings.ToLower(attr.Name.Local) == adId {
				state.creativeId = attr.Value
			}
		}
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case linearCase:
		state.injectVideoClicks = true
		state.injectTracker = true
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case videoClicksCase:
		state.injectVideoClicks = false
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
		if err = encoder.Flush(); err != nil {
			return err
		}
		trackerinjector.addClickTrackingEvent(outputXML, state.creativeId, false)
	case nonLinearAdsCase:
		state.injectTracker = true
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case trackingEventsCase:
		if state.isCreative {
			state.injectTracker = false
			if err = encoder.EncodeToken(token); err != nil {
				return err
			}
			if err = encoder.Flush(); err != nil {
				return err
			}
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, false)
		}
	default:
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	}
	return nil
}

func (trackerinjector *TrackerInjector) handleEndElement(token xml.EndElement, state *InjectionState, outputXML *strings.Builder, encoder *xml.Encoder) error {
	var err error
	switch token.Name.Local {
	case impressionCase:
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
		if err = encoder.Flush(); err != nil {
			return err
		}
		if !state.impressionTagFound {
			trackerinjector.addImpressionTrackingEvent(outputXML)
			state.impressionTagFound = true
		}
	case errorCase:
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
		if err = encoder.Flush(); err != nil {
			return err
		}
		if !state.errorTagFound {
			trackerinjector.addErrorTrackingEvent(outputXML)
			state.errorTagFound = true
		}
	case nonLinearAdsCase:
		if state.injectTracker {
			state.injectTracker = false
			if err = encoder.Flush(); err != nil {
				return err
			}
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, true)
			if !state.nonLinearTagFound && state.wrapperTagFound {
				trackerinjector.addNonLinearClickTrackingEvent(outputXML, state.creativeId, true)
			}
			if err = encoder.EncodeToken(token); err != nil {
				return err
			}
		}
	case linearCase:
		if state.injectVideoClicks {
			state.injectVideoClicks = false
			if err = encoder.Flush(); err != nil {
				return err
			}
			trackerinjector.addClickTrackingEvent(outputXML, state.creativeId, true)
		}
		if state.injectTracker {
			state.injectTracker = false
			if err = encoder.Flush(); err != nil {
				return err
			}
			trackerinjector.addTrackingEvent(outputXML, state.creativeId, true)
		}
		encoder.EncodeToken(token)
	case inlineCase, wrapperCase:
		state.wrapperTagFound = false
		state.inlineWrapperTagFound = true
		if err = encoder.Flush(); err != nil {
			return err
		}
		if !state.impressionTagFound {
			trackerinjector.addImpressionTrackingEvent(outputXML)
		}
		state.impressionTagFound = false
		if !state.errorTagFound {
			trackerinjector.addErrorTrackingEvent(outputXML)
		}
		state.errorTagFound = false
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case nonLinearCase:
		if err = encoder.Flush(); err != nil {
			return err
		}
		trackerinjector.addNonLinearClickTrackingEvent(outputXML, state.creativeId, false)
		state.nonLinearTagFound = true
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case companionCase:
		state.companionTagFound = true
		if err = encoder.Flush(); err != nil {
			return err
		}
		trackerinjector.addCompanionClickThroughEvent(outputXML, state.creativeId, false)
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case creativeCase:
		state.isCreative = false
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	case companionAdsCase:
		if !state.companionTagFound && state.wrapperTagFound {
			if err = encoder.Flush(); err != nil {
				return err
			}
			trackerinjector.addCompanionClickThroughEvent(outputXML, state.creativeId, true)
		}
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	default:
		if err = encoder.EncodeToken(token); err != nil {
			return err
		}
	}
	return nil
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
