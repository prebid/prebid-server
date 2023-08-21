package injector

import (
	"encoding/xml"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/metrics"
)

type VAST struct {
	Ads []Ad `xml:"Ad"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Ad struct {
	InLine  *InLine  `xml:",omitempty"`
	Wrapper *Wrapper `xml:",omitempty"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type InLine struct {
	Creatives   []Creative   `xml:"Creatives>Creative"`
	Impressions []Impression `xml:"Impression"`
	Errors      []Error      `xml:"Error"`
	Extra       []Node       `xml:",any"`
	Attributes  []xml.Attr   `xml:",any,attr"`
}

type Wrapper struct {
	Creatives []CreativeWrapper `xml:"Creatives>Creative"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Creative struct {
	Linear       *Linear       `xml:",omitempty"`
	NonLinearAds *NonLinearAds `xml:",omitempty"`
	CompanionAds *CompanionAds `xml:",omitempty"`
	Extra        []Node        `xml:",any"`
	Attributes   []xml.Attr    `xml:",any,attr"`
}

type WrapperCreative struct {
	TrackingEvents []Tracking `xml:"TrackingEvents>Tracking,omitempty"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type LinearCreative struct {
	TrackingEvents []Tracking     `xml:"TrackingEvents>Tracking,omitempty"`
	ClickThroughs  []ClickThrough `xml:"VideoClicks>ClickThrough,omitempty"`
	Extra          []Node         `xml:",any"`
	Attributes     []xml.Attr     `xml:",any,attr"`
}

type NonLinear struct {
	NonLinearClickThroughs []NonLinearClickThrough `xml:"NonLinearClickThrough,omitempty"`
	Extra                  []Node                  `xml:",any"`
	Attributes             []xml.Attr              `xml:",any,attr"`
}

type Companion struct {
	CompanionClickThroughs []CompanionClickThrough `xml:"CompanionClickThrough,omitempty"`
	Extra                  []Node                  `xml:",any"`
	Attributes             []xml.Attr              `xml:",any,attr"`
}

type CompanionClickThrough struct {
	URI        string     `xml:",cdata"`
	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type CompanionAds struct {
	Companion  []Companion `xml:"Companion,omitempty"`
	Extra      []Node      `xml:",any"`
	Attributes []xml.Attr  `xml:",any,attr"`
}

type Linear LinearCreative
type LinearWrapper WrapperCreative
type NonLinearAdsWrapper WrapperCreative

type NonLinearAds struct {
	TrackingEvents []Tracking  `xml:"TrackingEvents>Tracking,omitempty"`
	NonLinear      []NonLinear `xml:"NonLinear,omitempty"`
	Extra          []Node      `xml:",any"`
	Attributes     []xml.Attr  `xml:",any,attr"`
}

type CreativeWrapper struct {
	Linear       *LinearWrapper       `xml:",omitempty"`
	NonLinearAds *NonLinearAdsWrapper `xml:"NonLinearAds,omitempty"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type ClickThrough struct {
	URI        string     `xml:",cdata"`
	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type NonLinearClickThrough struct {
	URI        string     `xml:",cdata"`
	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Impression struct {
	URI        string     `xml:",cdata"`
	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Error struct {
	URI        string     `xml:",cdata"`
	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Tracking struct {
	Event string `xml:"event,attr"`
	URI   string `xml:",cdata"`

	Extra      []Node     `xml:",any"`
	Attributes []xml.Attr `xml:",any,attr"`
}

type Node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",cdata"`
	// Nodes   []Node     `xml:",any"`
}

func main() {

	events := map[string][]config.VASTEvent{}
	events["impression"] = []config.VASTEvent{{
		CreateElement: "impression",
		URLs:          []string{"http://impression.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}

	events["error"] = []config.VASTEvent{{
		CreateElement: "error",
		URLs:          []string{"http://error.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}

	events["tracking"] = []config.VASTEvent{{
		CreateElement: "tracking",
		URLs:          []string{"http://tracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}

	events["clicktracking"] = []config.VASTEvent{{
		CreateElement: "clicktracking",
		URLs:          []string{"http://clicktracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}

	events["nonlinearclicktracking"] = []config.VASTEvent{{
		CreateElement: "nonlinearclicktracking",
		URLs:          []string{"http://nonlinearclicktracking.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}

	events["companionclickthrough"] = []config.VASTEvent{{
		CreateElement: "companionclickthrough",
		URLs:          []string{"http://companionclickthrough.com?macro1=##PBS-BIDID##&macro2=##PBS-APPBUNDLE##&macro3=##PBS-DOMAIN##&macro4=##PBS-PUBDOMAIN##&macro5=##PBS-PAGEURL##&macro6=##PBS-ACCOUNTID##&macro7=##PBS-LIMITADTRACKING##&macro8=##PBS-GDPRCONSENT##&macro9=##PBS-MACRO-CUSTOMMACR1##&macro10=##PBS-BIDDER##&macro11=##PBS-INTEGRATION##&macro12=##PBS-VASTCRTID##&macro15=##PBS-AUCTIONID##&macro16=##PBS-CHANNEL##&macro17=##PBS-EVENTTYPE##&macro18=##PBS-VASTEVENT##"},
	}}
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
}

func NewTrackerInjector(replacer macros.Replacer, provider *macros.MacroProvider, events VASTEvents) *TrackerInjector {
	return &TrackerInjector{
		replacer: replacer,
		provider: provider,
		events:   events,
	}
}

func (builder *TrackerInjector) Build(vastXML string, NURL string) string {
	vast := VAST{}
	err := xml.Unmarshal([]byte(vastXML), &vast)
	if err != nil {
		err = fmt.Errorf("error parsing VAST XML. '%v'", err.Error())
		glog.Errorf(err.Error())
	}

	for _, ad := range vast.Ads {
		if ad.InLine != nil {
			for _, url := range builder.events.Impressions {
				url, _ = builder.replacer.Replace(url, builder.provider)
				ad.InLine.Impressions = append(ad.InLine.Impressions, Impression{URI: url})
			}

			for _, url := range builder.events.Errors {
				url, _ = builder.replacer.Replace(url, builder.provider)
				ad.InLine.Errors = append(ad.InLine.Errors, Error{URI: url})
			}

			for _, creative := range ad.InLine.Creatives {
				if creative.Linear != nil {
					for _, event := range builder.events.LinearTrackingEvents {
						for _, url := range event {
							url, _ := builder.replacer.Replace(url, builder.provider)
							creative.Linear.TrackingEvents = append(creative.Linear.TrackingEvents, Tracking{URI: url})
						}
					}

					for _, url := range builder.events.VideoClicks {
						url, _ := builder.replacer.Replace(url, builder.provider)

						creative.Linear.ClickThroughs = append(creative.Linear.ClickThroughs, ClickThrough{URI: url})
					}
				}

				if creative.NonLinearAds != nil {
					for _, event := range builder.events.LinearTrackingEvents {
						for _, url := range event {
							url, _ := builder.replacer.Replace(url, builder.provider)
							if err != nil {
								continue
							}
							creative.NonLinearAds.TrackingEvents = append(creative.NonLinearAds.TrackingEvents, Tracking{URI: url})
						}
					}
					for _, nl := range creative.NonLinearAds.NonLinear {
						for _, url := range builder.events.NonLinearClickTracking {
							url, _ := builder.replacer.Replace(url, builder.provider)
							nl.NonLinearClickThroughs = append(nl.NonLinearClickThroughs, NonLinearClickThrough{URI: url})
						}
					}
				}

				if creative.CompanionAds != nil {
					for _, cmp := range creative.CompanionAds.Companion {
						for _, url := range builder.events.CompanionClickThrough {
							url, _ := builder.replacer.Replace(url, builder.provider)
							cmp.CompanionClickThroughs = append(cmp.CompanionClickThroughs, CompanionClickThrough{URI: url})
						}
					}
				}
			}
		}
	}

	b, _ := xml.Marshal(vast)
	return string(b)
}
