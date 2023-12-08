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
	nonLineaerTagFound := false

	b := strings.NewReader(vastXML)
	p := xml.NewDecoder(b)

	for {
		t, err := p.RawToken()
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
				ti.provider.PopulateEventMacros(creativeId, "", "")

				for _, url := range ti.events.VideoClicks {
					outputXML.WriteString("<ClickTracking><![CDATA[")
					ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
					outputXML.WriteString("]]></ClickTracking>")
				}

				continue
			case "NonLinearAds":
				injectTracker = true
			case "TrackingEvents":
				if isCreative {
					injectTracker = false
					encoder.Flush()
					encoder.EncodeToken(tt)
					encoder.Flush()
					for typ, urls := range ti.events.LinearTrackingEvents {
						ti.provider.PopulateEventMacros(creativeId, "tracking", typ)
						for _, url := range urls {
							outputXML.WriteString("<Tracking event=\"")
							outputXML.WriteString(string(typ))
							outputXML.WriteString("\"><![CDATA[")
							ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
							outputXML.WriteString("]]></Tracking>")
						}
					}
					continue
				}
			}

		case xml.EndElement:
			switch tt.Name.Local {
			case "Impression":
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				fmt.Println()
				if !impressionTagFound {
					for _, url := range ti.events.Impressions {
						outputXML.WriteString("<Impression><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></Impression>")
					}
					impressionTagFound = true
				}
				continue
			case "Error":
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				if !errorTagFound {
					for _, url := range ti.events.Errors {
						outputXML.WriteString("<Error><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></Error>")

					}
					errorTagFound = true
				}
				continue
			case "NonLinearAds":
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					outputXML.WriteString("<TrackingEvents>")
					for typ, urls := range ti.events.LinearTrackingEvents {
						ti.provider.PopulateEventMacros(creativeId, "", typ)
						for _, url := range urls {

							outputXML.WriteString("<Tracking event=\"")
							outputXML.WriteString(typ)
							outputXML.WriteString("\"><![CDATA[")
							ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
							outputXML.WriteString("]]></Tracking>")
						}
					}
					outputXML.WriteString("</TrackingEvents>")

					if !nonLineaerTagFound && wrapperTagFound {
						outputXML.WriteString("<NonLinear>")
						ti.provider.PopulateEventMacros(creativeId, "", "")
						for _, url := range ti.events.NonLinearClickTracking {
							outputXML.WriteString("<NonLinearClickTracking><![CDATA[")
							ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
							outputXML.WriteString("]]></NonLinearClickTracking>")
						}
						outputXML.WriteString("</NonLinear>")
					}
					encoder.EncodeToken(tt)
				}
			case "Linear":
				if injectVideoClicks {
					injectVideoClicks = false
					encoder.Flush()
					outputXML.WriteString("<VideoClicks>")
					ti.provider.PopulateEventMacros(creativeId, "", "")

					for _, url := range ti.events.VideoClicks {

						outputXML.WriteString("<ClickTracking><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></ClickTracking>")
					}

					outputXML.WriteString("</VideoClicks>")

				}
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					outputXML.WriteString("<TrackingEvents>")
					for typ, urls := range ti.events.LinearTrackingEvents {
						ti.provider.PopulateEventMacros(creativeId, "", typ)
						for _, url := range urls {
							outputXML.WriteString("<Tracking event=\"")
							outputXML.WriteString(typ)
							outputXML.WriteString("\"><![CDATA[")
							ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
							outputXML.WriteString("]]></Tracking>")
						}
					}
					outputXML.WriteString("</TrackingEvents>")
				}
				encoder.EncodeToken(tt)
				continue
			case "InLine", "Wrapper":
				wrapperTagFound = false
				inlineWrapperTagFound = true
				encoder.Flush()

				if !impressionTagFound {
					for _, url := range ti.events.Impressions {
						outputXML.WriteString("<Impression><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></Impression>")
					}
				}
				impressionTagFound = false

				if !errorTagFound {
					for _, url := range ti.events.Errors {
						outputXML.WriteString("<Error><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></Error>")

					}
				}
				errorTagFound = false

				encoder.EncodeToken(tt)
			case "NonLinear":
				encoder.Flush()

				ti.provider.PopulateEventMacros(creativeId, "", "")
				for _, url := range ti.events.NonLinearClickTracking {
					outputXML.WriteString("<NonLinearClickTracking><![CDATA[")
					ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
					outputXML.WriteString("]]></NonLinearClickTracking>")
				}
				nonLineaerTagFound = true
				encoder.EncodeToken(tt)
			case "Companion":
				companionTagFound = true
				encoder.Flush()
				ti.provider.PopulateEventMacros(creativeId, "", "")
				for _, url := range ti.events.CompanionClickThrough {
					outputXML.WriteString("<CompanionClickThrough><![CDATA[")
					ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
					outputXML.WriteString("]]></CompanionClickThrough>")
				}
				encoder.EncodeToken(tt)
			case "Creative":
				isCreative = false
			case "CompanionAds":
				if !companionTagFound && wrapperTagFound {
					outputXML.WriteString("<Companion>")
					for _, url := range ti.events.CompanionClickThrough {
						outputXML.WriteString("<CompanionClickThrough><![CDATA[")
						ti.replacer.ReplaceBytes(&outputXML, url, ti.provider)
						outputXML.WriteString("]]></CompanionClickThrough>")
					}
					outputXML.WriteString("<Companion>")
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
