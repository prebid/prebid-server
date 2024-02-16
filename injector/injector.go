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

	injectTracker := false
	injectVideoClicks := false
	inlineWrapperTagFound := false
	wrapperTagFound := false
	impressionTagFound := false
	errorTagFound := false
	creativeId := ""
	isCreative := false
	companionTagFound := false
	nonLinearTagFound := false

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
			switch tt.Name.Local {
			case "Wrapper":
				wrapperTagFound = true
			case "Creative":
				isCreative = true
				for _, attr := range tt.Attr {
					if strings.ToLower(attr.Name.Local) == "adid" {
						creativeId = attr.Value
					}
				}
			case "Linear":
				injectVideoClicks = true
				injectTracker = true
			case "VideoClicks":
				injectVideoClicks = false
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				ti.addClickTrackingEvent(&outputXML, creativeId, false)

				continue
			case "NonLinearAds":
				injectTracker = true
			case "TrackingEvents":
				if isCreative {
					injectTracker = false
					encoder.Flush()
					encoder.EncodeToken(tt)
					encoder.Flush()
					ti.addTrackingEvent(&outputXML, creativeId, false)
					continue
				}
			}

		case xml.EndElement:
			switch tt.Name.Local {
			case "Impression":
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				if !impressionTagFound {
					ti.addImpressionTrackingEvent(&outputXML)
					impressionTagFound = true
				}
				continue
			case "Error":
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				if !errorTagFound {
					ti.addErrorTrackingEvent(&outputXML)
					errorTagFound = true
				}
				continue
			case "NonLinearAds":
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					ti.addTrackingEvent(&outputXML, creativeId, true)
					if !nonLinearTagFound && wrapperTagFound {
						ti.addNonLinearClickTrackingEvent(&outputXML, creativeId, true)
					}
					encoder.EncodeToken(tt)
				}
			case "Linear":
				if injectVideoClicks {
					injectVideoClicks = false
					encoder.Flush()
					ti.addClickTrackingEvent(&outputXML, creativeId, true)
				}
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					ti.addTrackingEvent(&outputXML, creativeId, true)
				}
				encoder.EncodeToken(tt)
				continue
			case "InLine", "Wrapper":
				wrapperTagFound = false
				inlineWrapperTagFound = true
				encoder.Flush()
				if !impressionTagFound {
					ti.addImpressionTrackingEvent(&outputXML)
				}
				impressionTagFound = false
				if !errorTagFound {
					ti.addErrorTrackingEvent(&outputXML)
				}
				errorTagFound = false
				encoder.EncodeToken(tt)
			case "NonLinear":
				encoder.Flush()
				ti.addNonLinearClickTrackingEvent(&outputXML, creativeId, false)
				nonLinearTagFound = true
				encoder.EncodeToken(tt)
			case "Companion":
				companionTagFound = true
				encoder.Flush()
				ti.addCompanionClickThroughEvent(&outputXML, creativeId, false)
				encoder.EncodeToken(tt)
			case "Creative":
				isCreative = false
			case "CompanionAds":
				if !companionTagFound && wrapperTagFound {
					encoder.Flush()
					ti.addCompanionClickThroughEvent(&outputXML, creativeId, true)
				}
			}
		case xml.CharData:
			tt2 := strings.Trim(string(tt), trimRunes)
			if len(tt2) != 0 {
				encoder.Flush()
				outputXML.WriteString("<![CDATA[")
				outputXML.WriteString(tt2)
				outputXML.WriteString("]]>")
				continue
			}
		}

		encoder.EncodeToken(t)
	}

	encoder.Flush()

	if !inlineWrapperTagFound {
		// 	// Todo log adapter.<bidder-name>.requests.badserverresponse metrics
		return vastXML
	}
	return outputXML.String()
}

func (ti *TrackerInjector) addTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString("<TrackingEvents>")
	}
	for typ, urls := range ti.events.LinearTrackingEvents {
		ti.writeTrackingEvent(urls, outputXML, "<Tracking event=\""+string(typ)+"\"><![CDATA[", "]]></Tracking>", creativeId, typ, "tracking")
	}
	if addParentTag {
		outputXML.WriteString("</TrackingEvents>")
	}
}

func (ti *TrackerInjector) addClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString("<VideoClicks>")
	}
	ti.writeTrackingEvent(ti.events.VideoClicks, outputXML, "<ClickTracking><![CDATA[", "]]></ClickTracking>", creativeId, "", "")
	if addParentTag {
		outputXML.WriteString("</VideoClicks>")
	}
}

func (ti *TrackerInjector) addImpressionTrackingEvent(outputXML *strings.Builder) {
	ti.writeTrackingEvent(ti.events.Impressions, outputXML, "<Impression><![CDATA[", "]]></Impression>", "", "", "")
}

func (ti *TrackerInjector) addErrorTrackingEvent(outputXML *strings.Builder) {
	ti.writeTrackingEvent(ti.events.Errors, outputXML, "<Error><![CDATA[", "]]></Error>", "", "", "")
}

func (ti *TrackerInjector) addNonLinearClickTrackingEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString("<NonLinear>")
	}
	ti.writeTrackingEvent(ti.events.NonLinearClickTracking, outputXML, "<NonLinearClickTracking><![CDATA[", "]]></NonLinearClickTracking>", creativeId, "", "")
	if addParentTag {
		outputXML.WriteString("</NonLinear>")
	}
}

func (ti *TrackerInjector) addCompanionClickThroughEvent(outputXML *strings.Builder, creativeId string, addParentTag bool) {
	if addParentTag {
		outputXML.WriteString("<Companion>")
	}
	ti.writeTrackingEvent(ti.events.CompanionClickThrough, outputXML, "<CompanionClickThrough><![CDATA[", "]]></CompanionClickThrough>", creativeId, "", "")
	if addParentTag {
		outputXML.WriteString("</Companion>")
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
