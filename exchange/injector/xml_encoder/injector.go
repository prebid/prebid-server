package injector

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/metrics"
)

type VASTEvents struct {
	Errors                  []string
	Impressions             []string
	VideoClicks             []string
	NonLinearClickTracking  []string
	CompanionClickThrough   []string
	LinearTrackingEvents    map[string][]string //TODO:this can be normalise to array[{type,url}]
	NonLinearTrackingEvents map[string][]string
	CompanionTrackingEvents map[string][]string
}

var trimRunes = "\t\r\b\n "

type TrackerInjector struct {
	replacer macros.Replacer
	events   VASTEvents
	me       metrics.MetricsEngine
	provider *macros.MacroProvider
}

func NewTrackerInjector(replacer macros.Replacer, provider *macros.MacroProvider, events VASTEvents) *TrackerInjector {
	return &TrackerInjector{
		replacer: replacer,
		provider: provider,
		events:   events,
	}
}

func (builder *TrackerInjector) Build(vastXML string, NURL string) string {
	var outputXML strings.Builder
	encoder := xml.NewEncoder(&outputXML)

	injectTracker := false
	injectVideoClicks := false
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
			case "Linear":
				injectVideoClicks = true
				injectTracker = true
			case "VideoClicks":
				injectVideoClicks = false
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				builder.provider.PopulateEventMacros("creativeId", "", "")

				for _, url := range builder.events.VideoClicks {
					builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)

					outputXML.WriteString("<ClickTracking><![CDATA[")
					outputXML.WriteString(url)
					outputXML.WriteString("]]></ClickTracking>")
				}

				continue
			case "NonLinearAds":
				injectTracker = true
			case "TrackingEvents":
				injectTracker = false
				encoder.Flush()
				encoder.EncodeToken(tt)
				encoder.Flush()
				for typ, urls := range builder.events.LinearTrackingEvents {
					builder.provider.PopulateEventMacros("creativeId", "tracking", typ)
					for _, url := range urls {
						outputXML.WriteString("<Tracking event=\"")
						outputXML.WriteString(string(typ))
						outputXML.WriteString("\"><![CDATA[")
						builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
						outputXML.WriteString("]]></Tracking>")
					}
				}
				continue
			}

		case xml.EndElement:
			switch tt.Name.Local {
			case "NonLinearAds":
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					outputXML.WriteString("<TrackingEvents>")
					for typ, urls := range builder.events.LinearTrackingEvents {
						builder.provider.PopulateEventMacros("creativeId", "", typ)
						for _, url := range urls {

							outputXML.WriteString("<Tracking event=\"")
							outputXML.WriteString(typ)
							outputXML.WriteString("\"><![CDATA[")
							builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
							outputXML.WriteString("]]></Tracking>")
						}
					}
					outputXML.WriteString("</TrackingEvents>")
					encoder.EncodeToken(tt)
				}
			case "Linear":
				if injectVideoClicks {
					injectVideoClicks = false
					encoder.Flush()
					outputXML.WriteString("<VideoClicks>")
					builder.provider.PopulateEventMacros("creativeId", "", "")

					for _, url := range builder.events.VideoClicks {

						outputXML.WriteString("<ClickTracking><![CDATA[")
						builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
						outputXML.WriteString("]]></ClickTracking>")
					}

					outputXML.WriteString("</VideoClicks>")
					encoder.EncodeToken(tt)
				}
				if injectTracker {
					injectTracker = false
					encoder.Flush()
					outputXML.WriteString("<TrackingEvents>")
					for typ, urls := range builder.events.LinearTrackingEvents {
						builder.provider.PopulateEventMacros("creativeId", "", typ)
						for _, url := range urls {
							outputXML.WriteString("<Tracking event=\"")
							outputXML.WriteString(typ)
							outputXML.WriteString("\"><![CDATA[")
							builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
							outputXML.WriteString("]]></Tracking>")
						}
					}
					outputXML.WriteString("</TrackingEvents>")
					encoder.EncodeToken(tt)
				}

			case "InLine", "Wrapper":
				encoder.Flush()
				for _, url := range builder.events.Impressions {
					outputXML.WriteString("<Impression><![CDATA[")
					builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
					outputXML.WriteString("]]></Impression>")
				}

				for _, url := range builder.events.Errors {
					outputXML.WriteString("<Error><![CDATA[")
					builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
					outputXML.WriteString("]]></Error>")

				}
				encoder.EncodeToken(tt)
			case "NonLinear":
				encoder.Flush()

				builder.provider.PopulateEventMacros("creativeId", "", "")
				for _, url := range builder.events.NonLinearClickTracking {
					outputXML.WriteString("<NonLinearClickTracking><![CDATA[")
					builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
					outputXML.WriteString("]]></NonLinearClickTracking>")
				}

				encoder.EncodeToken(tt)
			case "Companion":
				encoder.Flush()
				builder.provider.PopulateEventMacros("creativeId", "", "")
				for _, url := range builder.events.CompanionClickThrough {
					outputXML.WriteString("<CompanionClickThrough><![CDATA[")
					builder.replacer.ReplaceBytes(&outputXML, url, builder.provider)
					outputXML.WriteString("]]></CompanionClickThrough>")

				}
				encoder.EncodeToken(tt)
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
	return outputXML.String()
}
