package injector

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"sort"
	"strings"
	"sync"

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

var (
	creativeId = regexp.MustCompile(`^<Creative.*id\s*=\s*"([a-z,A-Z,0-9]+)".*>$`)
)

const (
	errorVast = `<VAST version=\"3.0\"><Ad><Wrapper>
	<AdSystem>prebid.org wrapper</AdSystem>
	<VASTAdTagURI><![CDATA[" %s "]]></VASTAdTagURI>
	<Creatives></Creatives>
	</Wrapper></Ad></VAST>`
)

const (
	creativesStartTag         = "<Creatives>"
	trackingEventsTagStartTag = "<TrackingEvents>"
	trackingEventsTagEndTag   = "</TrackingEvents>"
	videoClicksStartTag       = "<VideoClicks>"
	videoClicksEndTag         = "</VideoClicks>"
	nonLinearStartTag         = "<NonLinear>"
	nonLinearEndTag           = "</NonLinear>"
	linearEndTag              = "</Linear>"
	nonLinearAdsEndTag        = "</NonLinearAds>"
	wrapperEndTag             = "</Wrapper>"
	wrapperStartTag           = "<Wrapper>"
	inLineEndTag              = "</InLine>"
	adSystemEndTag            = "</AdSystem>"
	creativeEndTag            = "</Creative>"
	companionStartTag         = "<Companion>"
	companionEndTag           = "</Companion>"
	impressionEndTag          = "</Impression>"
	companionAdsEndTag        = "</CompanionAds>"
	adElementEndTag           = "</Ad>"
	errorEndTag               = "</Error>"
)

type Injector interface {
	Build(vastXML, nURL string) string
}

type TrackerInjector struct {
	replacer macros.Replacer
	events   VASTEvents
	me       metrics.MetricsEngine
	provider *macros.MacroProvider
}

func NewTrackerInjector(replacer macros.Replacer, provider *macros.MacroProvider, events VASTEvents) Injector {
	return &TrackerInjector{
		replacer: replacer,
		provider: provider,
		events:   events,
	}
}

type Ad struct {
	WrapperInlineEndIndex int
	ImpressionEndIndex    int
	ErrorEndIndex         int
	Creatives             []Creative
}

type Creative struct {
	Linear       *Linear
	NonLinearAds *NonLinearAds
	CompanionAds *CompanionAds
	CreativeID   string
}

type NonLinearAds struct {
	TrackingEvent     int
	NonLinears        []int
	NonLinearAdsIndex int
}

type CompanionAds struct {
	Companion         []int
	CompanionAdsIndex int
}

type Linear struct {
	VideoClick     int
	TrackingEvent  int
	LinearEndIndex int
}

// Preallocate a strings.Builder and a byte slice for the final result
var builderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// pair maintains the index and tag to be injected
type pair struct {
	pos         int
	tag         string
	wrapParent  bool
	eventMacros map[string]string
}

func (builder *TrackerInjector) Build(vastXML string, NURL string) string {

	ads := parseVastXML([]byte(vastXML))
	pairs := builder.buildPairs(ads)
	//sort all events position
	sort.SliceStable(pairs[:], func(i, j int) bool {
		return pairs[i].pos < pairs[j].pos
	})

	// Reuse a preallocated strings.Builder
	buf := builderPool.Get().(*strings.Builder)
	buf.Reset()
	defer builderPool.Put(buf)
	offset := 0
	for i := range pairs {
		if offset != pairs[i].pos {
			buf.WriteString(vastXML[offset:pairs[i].pos])
			offset = pairs[i].pos
		}
		builder.addEvent(buf, pairs[i])
	}
	buf.WriteString(vastXML[offset:])
	return buf.String()
}

func parseVastXML(vastXML []byte) []Ad {

	var (
		vastTags = make([]Ad, 0, 10)
	)

	ad := Ad{}
	creative := Creative{}
	trackingEventEndIndex := 0
	videoClick := 0
	nonLinearAds := make([]int, 0, 10)
	companions := make([]int, 0, 10)
	//length := len(vastXML)

	dec := xml.NewDecoder(bytes.NewReader(vastXML))

	for {
		t, err := dec.RawToken()
		if err != nil {
			break
		}

		switch t := t.(type) {
		case xml.EndElement:
			handleTag(string(t.Name.Local), &ad, &creative, &vastTags, &trackingEventEndIndex, &videoClick, &nonLinearAds, &companions, int(dec.InputOffset()))

		}

	}
	return vastTags
}

func handleTag(tag string, ad *Ad, creative *Creative, vastTags *[]Ad, trackingEventEndIndex *int, videoClick *int, nonLinearAds *[]int, companions *[]int, index int) {
	switch tag {
	case "Ad":
		*vastTags = append(*vastTags, *ad)
		*ad = Ad{}
	case "Impression":
		ad.ImpressionEndIndex = index
	case "Error":
		ad.ErrorEndIndex = index
	case "Creative":
		ad.Creatives = append(ad.Creatives, *creative)
		*creative = Creative{}
	case "InLine", "Wrapper":
		ad.WrapperInlineEndIndex = index
	case "TrackingEvents":
		*trackingEventEndIndex = index
	case "VideoClicks":
		*videoClick = index
	case "NonLinear":
		*nonLinearAds = append(*nonLinearAds, index)
	case "NonLinearAds":
		creative.NonLinearAds = &NonLinearAds{
			TrackingEvent:     *trackingEventEndIndex,
			NonLinearAdsIndex: index,
			NonLinears:        *nonLinearAds,
		}
	case "Linear":
		creative.Linear = &Linear{
			TrackingEvent:  *trackingEventEndIndex,
			LinearEndIndex: index,
			VideoClick:     *videoClick,
		}
		*videoClick = 0
		*trackingEventEndIndex = 0
	case "Companion":
		*companions = append(*companions, index)
	case "CompanionAds":
		creative.CompanionAds = &CompanionAds{
			CompanionAdsIndex: index,
			Companion:         *companions,
		}
	}
}

func (builder *TrackerInjector) buildPairs(vastTags []Ad) []pair {
	pairs := make([]pair, 0, len(vastTags)*4)
	for _, tag := range vastTags {
		if tag.ImpressionEndIndex != 0 {
			pairs = append(pairs, pair{pos: tag.ImpressionEndIndex, tag: "impression"})
		} else {
			pairs = append(pairs, pair{pos: tag.WrapperInlineEndIndex, tag: "impression", wrapParent: true})
		}
		if tag.ErrorEndIndex != 0 {
			pairs = append(pairs, pair{pos: tag.ErrorEndIndex, tag: "error"})
		} else {
			pairs = append(pairs, pair{pos: tag.WrapperInlineEndIndex, tag: "error", wrapParent: true})
		}

		for _, creative := range tag.Creatives {
			if creative.Linear != nil {
				if creative.Linear.TrackingEvent == 0 {
					pairs = append(pairs, pair{pos: creative.Linear.LinearEndIndex, tag: "tracking", wrapParent: true})
				} else {
					pairs = append(pairs, pair{pos: creative.Linear.TrackingEvent, tag: "tracking"})
				}

				if creative.Linear.VideoClick == 0 {
					pairs = append(pairs, pair{pos: creative.Linear.LinearEndIndex, tag: "clicktracking", wrapParent: true})
				} else {
					pairs = append(pairs, pair{pos: creative.Linear.VideoClick, tag: "clicktracking"})
				}
			}

			if creative.NonLinearAds != nil {
				if creative.NonLinearAds.TrackingEvent == 0 {
					pairs = append(pairs, pair{pos: creative.NonLinearAds.NonLinearAdsIndex, tag: "tracking", wrapParent: true})
				} else {
					pairs = append(pairs, pair{pos: creative.NonLinearAds.TrackingEvent, tag: "tracking"})
				}

				if len(creative.NonLinearAds.NonLinears) == 0 {
					pairs = append(pairs, pair{pos: creative.NonLinearAds.NonLinearAdsIndex, tag: "nonlinearclicktracking", wrapParent: true})
				} else {
					for _, nonLinear := range creative.NonLinearAds.NonLinears {
						pairs = append(pairs, pair{pos: nonLinear, tag: "nonlinearclicktracking"})
					}
				}
			}

			if creative.CompanionAds != nil {
				if len(creative.CompanionAds.Companion) == 0 {
					pairs = append(pairs, pair{pos: creative.CompanionAds.CompanionAdsIndex, tag: "companionclickthrough", wrapParent: true})
				} else {
					for _, companion := range creative.CompanionAds.Companion {
						pairs = append(pairs, pair{pos: companion, tag: "companionclickthrough"})
					}
				}
			}
		}
	}

	return pairs
}

func (builder *TrackerInjector) addEvent(buf *strings.Builder, pair pair) {

	switch pair.tag {
	case "impression":
		for _, url := range builder.events.Impressions {
			buf.WriteString(`<Impression><![CDATA[`)
			builder.replacer.ReplaceBytes(buf, url, builder.provider)
			buf.WriteString(`]]></Impression>`)
		}

	case "error":
		for _, url := range builder.events.Errors {
			buf.WriteString(`<Error><![CDATA[`)
			builder.replacer.ReplaceBytes(buf, url, builder.provider)
			buf.WriteString(`]]></Error>`)
		}
	case "tracking":
		if pair.wrapParent {
			buf.WriteString("<TrackingEvents>")
		}
		for typ, urls := range builder.events.LinearTrackingEvents {
			builder.provider.PopulateEventMacros("creativeId", "lineartracking", string(typ))
			for _, url := range urls {
				buf.WriteString(`<Tracking event="`)
				buf.WriteString(string(typ))
				buf.WriteString(`"><![CDATA[`)
				builder.replacer.ReplaceBytes(buf, url, builder.provider)
				buf.WriteString(`]]></Tracking>`)
			}
		}

		if pair.wrapParent {
			buf.WriteString("</TrackingEvents>")
		}
	case "nonlinearclicktracking":
		builder.provider.PopulateEventMacros("creativeId", "nonlinearclicktracking", "")
		for _, url := range builder.events.NonLinearClickTracking {
			buf.WriteString(`<NonLinearClickTracking><![CDATA[`)
			builder.replacer.ReplaceBytes(buf, url, builder.provider)
			buf.WriteString(`]]></NonLinearClickTracking>`)
		}
	case "clicktracking":
		if pair.wrapParent {
			buf.WriteString("<VideoClicks>")
		}
		builder.provider.PopulateEventMacros("creativeId", "", "")

		for _, url := range builder.events.VideoClicks {

			buf.WriteString(`<ClickTracking><![CDATA[`)
			builder.replacer.ReplaceBytes(buf, url, builder.provider)
			buf.WriteString(`]]></ClickTracking>`)
		}
		if pair.wrapParent {
			buf.WriteString("</VideoClicks>")
		}
	case "companionclickthrough":
		if pair.wrapParent {
			buf.WriteString("<Companion>")
		}
		builder.provider.PopulateEventMacros("creativeId", "nonlinearclicktracking", "")

		for _, url := range builder.events.CompanionClickThrough {
			buf.WriteString(`<CompanionClickThrough><![CDATA[`)
			builder.replacer.ReplaceBytes(buf, url, builder.provider)
			buf.WriteString(`]]></CompanionClickThrough>`)
		}
		if pair.wrapParent {
			buf.WriteString("</Companion>")
		}
	}
}
