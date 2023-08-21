package injector

import (
	"github.com/beevik/etree"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/metrics"
)

type Injector interface {
	Build(vastXML, nURL string) string
}
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

type TrackerInjector struct {
	replacer macros.Replacer
	events   VASTEvents
	me       metrics.MetricsEngine
	provider *macros.MacroProvider
	doc      *etree.Document
}

func NewTrackerInjector(replacer macros.Replacer, provider *macros.MacroProvider, events VASTEvents) Injector {
	return &TrackerInjector{
		doc:      etree.NewDocument(),
		provider: provider,
		events:   events,
		replacer: replacer,
	}
}

func (builder *TrackerInjector) Build(vastXML string, NURL string) string {
	builder.doc.ReadFromString(vastXML)

	inlines := builder.doc.FindElements("VAST/Ad/InLine")
	if len(inlines) != 0 {
		for _, inline := range inlines {
			for _, url := range builder.events.Impressions {
				url, _ := builder.replacer.Replace(url, builder.provider)
				impression := inline.CreateElement("Impression")
				impression.CreateAttr("id", "")
				impression.CreateCData(url)
				inline.AddChild(impression)
			}
			for _, url := range builder.events.Errors {
				url, _ := builder.replacer.Replace(url, builder.provider)

				error := inline.CreateElement("error")
				error.CreateCData(url)
				inline.AddChild(error)
			}

			creatives := inline.FindElements("Creatives/Creative")
			for _, creative := range creatives {
				linear := creative.SelectElements("Linear")
				for _, li := range linear {
					trackingEvents := li.SelectElement("TrackingEvents")
					if trackingEvents == nil {
						trackingEvents = li.CreateElement("TrackingEvents")
						li.AddChild(trackingEvents)
					}
					for _, urls := range builder.events.LinearTrackingEvents {
						builder.provider.PopulateEventMacros("creativeID", "", "")
						for _, url := range urls {
							url, err := builder.replacer.Replace(url, builder.provider)
							if err != nil {
								continue
							}
							tracking := trackingEvents.CreateElement("Tracking")
							tracking.CreateCData(url)
							trackingEvents.AddChild(tracking)
						}
					}
					videoClicks := li.SelectElement("VideoClicks")
					if videoClicks == nil {
						videoClicks = li.CreateElement("VideoClicks")
					}
					builder.provider.PopulateEventMacros("creativeId", "", "")

					for _, url := range builder.events.VideoClicks {
						url, _ := builder.replacer.Replace(url, builder.provider)
						clicktracking := videoClicks.CreateElement("ClickTracking")
						clicktracking.CreateCData(url)
						videoClicks.AddChild(clicktracking)
					}
				}

				nonlinear := creative.SelectElement("NonLinearAds")
				if nonlinear != nil {
					trackingEvents := nonlinear.SelectElement("TrackingEvents")
					if trackingEvents == nil {
						trackingEvents = nonlinear.CreateElement("TrackingEvents")
						nonlinear.AddChild(trackingEvents)
					}
					builder.provider.PopulateEventMacros("creativeID", "", "")

					for _, url := range builder.events.NonLinearClickTracking {
						url, _ := builder.replacer.Replace(url, builder.provider)
						tracking := trackingEvents.CreateElement("Tracking")
						tracking.CreateCData(url)
						trackingEvents.AddChild(tracking)
					}

					nlis := nonlinear.SelectElements("NonLinear")
					for _, nli := range nlis {
						for _, url := range builder.events.NonLinearClickTracking {
							builder.provider.PopulateEventMacros("creativeId", "", "")
							url, _ := builder.replacer.Replace(url, builder.provider)
							nlct := nli.CreateElement("NonLinearClickTracking")
							nlct.CreateCData(url)
							nli.AddChild(nlct)
						}
					}
				}
				cds := creative.SelectElement("CompanionAds")
				if cds != nil {
					companions := creative.SelectElements("Companion")
					for _, companion := range companions {
						builder.provider.PopulateEventMacros("creativeId", "", "")
						for _, url := range builder.events.CompanionClickThrough {
							url, _ := builder.replacer.Replace(url, builder.provider)
							ele := companion.CreateElement("CompanionClickThrough")
							ele.CreateCData(url)
							companion.AddChild(ele)
						}
					}
				}
			}
		}
	}

	b, _ := builder.doc.WriteToBytes()
	return string(b)
}
