package floors

import (
	"fmt"
	"math/bits"
	"regexp"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	SiteDomain          string = "siteDomain"
	PubDomain           string = "pubDomain"
	Domain              string = "domain"
	Bundle              string = "bundle"
	Channel             string = "channel"
	MediaType           string = "mediaType"
	Size                string = "size"
	GptSlot             string = "gptSlot"
	AdUnitCode          string = "adUnitCode"
	Country             string = "country"
	DeviceType          string = "deviceType"
	Tablet              string = "tablet"
	Desktop             string = "desktop"
	Phone               string = "phone"
	BannerMedia         string = "banner"
	VideoMedia          string = "video"
	VideoOutstreamMedia string = "video-outstream"
	AudioMedia          string = "audio"
	NativeMedia         string = "native"
)

// getFloorCurrency returns floors currency provided in floors JSON,
// if currency is not provided then defaults to USD
func getFloorCurrency(floorExt *openrtb_ext.PriceFloorRules) string {
	floorCur := defaultCurrency
	if floorExt != nil && floorExt.Data != nil {
		if floorExt.Data.Currency != "" {
			floorCur = floorExt.Data.Currency
		}

		if len(floorExt.Data.ModelGroups) > 0 && floorExt.Data.ModelGroups[0].Currency != "" {
			floorCur = floorExt.Data.ModelGroups[0].Currency
		}
	}

	return floorCur
}

// getMinFloorValue returns floorMin and floorMinCur,
// values provided in impression extension are considered over floors JSON.
func getMinFloorValue(floorExt *openrtb_ext.PriceFloorRules, imp *openrtb_ext.ImpWrapper, conversions currency.Conversions) (float64, string, error) {
	var err error
	var rate float64
	var floorCur string
	floorMin := roundToFourDecimals(floorExt.FloorMin)
	floorMinCur := floorExt.FloorMinCur

	impFloorMin, impFloorCur, err := getFloorMinAndCurFromImp(imp)
	if err == nil {
		if impFloorMin > 0.0 {
			floorMin = impFloorMin
		}
		if impFloorCur != "" {
			floorMinCur = impFloorCur
		}

		floorCur = getFloorCurrency(floorExt)
		if floorMin > 0.0 && floorMinCur != "" {
			if floorExt.FloorMinCur != "" && impFloorCur != "" && floorExt.FloorMinCur != impFloorCur {
				glog.Warning("FloorMinCur are different in floorExt and ImpExt")
			}
			if floorCur != "" && floorMinCur != floorCur {
				rate, err = conversions.GetRate(floorMinCur, floorCur)
				floorMin = rate * floorMin
			}
		}
		floorMin = roundToFourDecimals(floorMin)
	}
	if err != nil {
		return floorMin, floorCur, fmt.Errorf("Error in getting FloorMin value : '%v'", err.Error())
	} else {
		return floorMin, floorCur, err
	}
}

// getFloorMinAndCurFromImp returns floorMin and floorMinCur from impression extension
func getFloorMinAndCurFromImp(imp *openrtb_ext.ImpWrapper) (float64, string, error) {
	var floorMin float64
	var floorMinCur string

	impExt, err := imp.GetImpExt()
	if impExt != nil {
		impExtPrebid := impExt.GetPrebid()
		if impExtPrebid != nil && impExtPrebid.Floors != nil {
			if impExtPrebid.Floors.FloorMin > 0.0 {
				floorMin = impExtPrebid.Floors.FloorMin
			}

			if impExtPrebid.Floors.FloorMinCur != "" {
				floorMinCur = impExtPrebid.Floors.FloorMinCur
			}
		}
	}
	return floorMin, floorMinCur, err
}

// updateImpExtWithFloorDetails updates floors related details into imp.ext.prebid.floors
func updateImpExtWithFloorDetails(imp *openrtb_ext.ImpWrapper, matchedRule string, floorRuleVal, floorVal float64) error {
	impExt, err := imp.GetImpExt()
	if err != nil {
		return err
	}
	extImpPrebid := impExt.GetPrebid()
	if extImpPrebid == nil {
		extImpPrebid = &openrtb_ext.ExtImpPrebid{}
	}
	extImpPrebid.Floors = &openrtb_ext.ExtImpPrebidFloors{
		FloorRule:      matchedRule,
		FloorRuleValue: floorRuleVal,
		FloorValue:     floorVal,
	}
	impExt.SetPrebid(extImpPrebid)
	return err
}

// selectFloorModelGroup selects one modelgroup based on modelweight out of multiple modelgroups, if provided into floors JSON.
func selectFloorModelGroup(modelGroups []openrtb_ext.PriceFloorModelGroup, f func(int) int) []openrtb_ext.PriceFloorModelGroup {
	totalModelWeight := 0
	for i := 0; i < len(modelGroups); i++ {
		if modelGroups[i].ModelWeight == nil {
			modelGroups[i].ModelWeight = new(int)
			*modelGroups[i].ModelWeight = 1
		}
		totalModelWeight += *modelGroups[i].ModelWeight

	}

	sort.SliceStable(modelGroups, func(i, j int) bool {
		if modelGroups[i].ModelWeight != nil && modelGroups[j].ModelWeight != nil {
			return *modelGroups[i].ModelWeight < *modelGroups[j].ModelWeight
		}
		return false
	})

	winWeight := f(totalModelWeight + 1)
	for i, modelGroup := range modelGroups {
		winWeight -= *modelGroup.ModelWeight
		if winWeight <= 0 {
			modelGroups[0], modelGroups[i] = modelGroups[i], modelGroups[0]
			return modelGroups[:1]
		}

	}
	return modelGroups[:1]
}

// shouldSkipFloors returns flag to decide skipping of floors singalling based on skipRate provided
func shouldSkipFloors(ModelGroupsSkipRate, DataSkipRate, RootSkipRate int, f func(int) int) bool {
	skipRate := 0

	if ModelGroupsSkipRate > 0 {
		skipRate = ModelGroupsSkipRate
	} else if DataSkipRate > 0 {
		skipRate = DataSkipRate
	} else {
		skipRate = RootSkipRate
	}

	if skipRate == 0 {
		return false
	}
	return skipRate >= f(skipRateMax+1)
}

// findRule prepares rule combinations based on schema dimensions provided in floors data, request values associated with these fields and
// does matching with rules provided in floors data and returns matched rule
func findRule(ruleValues map[string]float64, delimiter string, desiredRuleKey []string) (string, bool) {
	ruleKeys := prepareRuleCombinations(desiredRuleKey, delimiter)
	for i := 0; i < len(ruleKeys); i++ {
		if _, ok := ruleValues[ruleKeys[i]]; ok {
			return ruleKeys[i], true
		}
	}
	return "", false
}

// createRuleKey prepares rule keys based on schema dimension and values present in request
func createRuleKey(floorSchema openrtb_ext.PriceFloorSchema, request *openrtb_ext.RequestWrapper, imp *openrtb_ext.ImpWrapper) []string {
	var ruleKeys []string

	for _, field := range floorSchema.Fields {
		value := catchAll
		switch field {
		case MediaType:
			value = getMediaType(imp.Imp)
		case Size:
			value = getSizeValue(imp.Imp)
		case Domain:
			value = getDomain(request)
		case SiteDomain:
			value = getSiteDomain(request)
		case Bundle:
			value = getBundle(request)
		case PubDomain:
			value = getPublisherDomain(request)
		case Country:
			value = getDeviceCountry(request)
		case DeviceType:
			value = getDeviceType(request)
		case Channel:
			value = getChannelName(request)
		case GptSlot:
			value = getGptSlot(imp)
		case AdUnitCode:
			value = getAdUnitCode(imp)
		}
		ruleKeys = append(ruleKeys, value)
	}
	return ruleKeys
}

// getDeviceType returns device type provided into request
func getDeviceType(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.Device == nil || len(request.Device.UA) == 0 {
		return value
	}
	if isMobileDevice(request.Device.UA) {
		value = Phone
	} else if isTabletDevice(request.Device.UA) {
		value = Tablet
	} else {
		value = Desktop
	}
	return value
}

// getDeviceCountry returns device country provided into request
func getDeviceCountry(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.Device != nil && request.Device.Geo != nil {
		value = request.Device.Geo.Country
	}
	return value
}

// getMediaType returns media type for give impression
func getMediaType(imp *openrtb2.Imp) string {
	value := catchAll
	formatCount := 0

	if imp.Banner != nil {
		formatCount++
		value = BannerMedia
	}
	if imp.Video != nil && imp.Video.Placement != 1 {
		formatCount++
		value = VideoOutstreamMedia
	}
	if imp.Video != nil && imp.Video.Placement == 1 {
		formatCount++
		value = VideoMedia
	}
	if imp.Audio != nil {
		formatCount++
		value = AudioMedia
	}
	if imp.Native != nil {
		formatCount++
		value = NativeMedia
	}

	if formatCount > 1 {
		return catchAll
	}
	return value
}

// getSizeValue returns size for given media type in WxH format
func getSizeValue(imp *openrtb2.Imp) string {
	size := catchAll
	width := int64(0)
	height := int64(0)

	if imp.Banner != nil {
		width, height = getBannerSize(imp)
	} else if imp.Video != nil {
		width = ptrutil.ValueOrDefault(imp.Video.W)
		height = ptrutil.ValueOrDefault(imp.Video.H)
	}

	if width != 0 && height != 0 {
		size = fmt.Sprintf("%dx%d", width, height)
	}
	return size
}

// getBannerSize returns width and height for given banner impression
func getBannerSize(imp *openrtb2.Imp) (int64, int64) {
	width := int64(0)
	height := int64(0)

	if len(imp.Banner.Format) == 1 {
		return imp.Banner.Format[0].W, imp.Banner.Format[0].H
	} else if len(imp.Banner.Format) > 1 {
		return width, height
	} else if imp.Banner.W != nil && imp.Banner.H != nil {
		width = *imp.Banner.W
		height = *imp.Banner.H
	}
	return width, height
}

// getDomain returns domain provided into site or app object
func getDomain(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.Site != nil {
		if len(request.Site.Domain) > 0 {
			value = request.Site.Domain
		} else if request.Site.Publisher != nil && len(request.Site.Publisher.Domain) > 0 {
			value = request.Site.Publisher.Domain
		}
	} else if request.App != nil {
		if len(request.App.Domain) > 0 {
			value = request.App.Domain
		} else if request.App.Publisher != nil && len(request.App.Publisher.Domain) > 0 {
			value = request.App.Publisher.Domain
		}
	}
	return value
}

// getSiteDomain  returns domain provided into site object
func getSiteDomain(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.Site != nil && len(request.Site.Domain) > 0 {
		value = request.Site.Domain
	} else if request.App != nil && len(request.App.Domain) > 0 {
		value = request.App.Domain
	}
	return value
}

// getPublisherDomain returns publisher domain provided into site or app object
func getPublisherDomain(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.Site != nil && request.Site.Publisher != nil && len(request.Site.Publisher.Domain) > 0 {
		value = request.Site.Publisher.Domain
	} else if request.App != nil && request.App.Publisher != nil && len(request.App.Publisher.Domain) > 0 {
		value = request.App.Publisher.Domain
	}
	return value
}

// getBundle returns app bundle type
func getBundle(request *openrtb_ext.RequestWrapper) string {
	value := catchAll
	if request.App != nil && len(request.App.Bundle) > 0 {
		value = request.App.Bundle
	}
	return value
}

// getGptSlot returns gptSlot
func getGptSlot(imp *openrtb_ext.ImpWrapper) string {
	value := catchAll

	impExt, err := imp.GetImpExt()
	if err == nil {
		extData := impExt.GetData()
		if extData != nil {
			if extData.AdServer != nil && extData.AdServer.Name == "gam" {
				gptSlot := extData.AdServer.AdSlot
				if gptSlot != "" {
					value = gptSlot
				}
			} else if extData.PbAdslot != "" {
				value = extData.PbAdslot
			}
		}
	}
	return value
}

// getChannelName returns channel name
func getChannelName(bidRequest *openrtb_ext.RequestWrapper) string {
	reqExt, err := bidRequest.GetRequestExt()
	if err == nil && reqExt != nil {
		prebidExt := reqExt.GetPrebid()
		if prebidExt != nil && prebidExt.Channel != nil {
			return prebidExt.Channel.Name
		}
	}
	return catchAll
}

// getAdUnitCode returns adUnit code
func getAdUnitCode(imp *openrtb_ext.ImpWrapper) string {
	adUnitCode := catchAll

	impExt, err := imp.GetImpExt()
	if err == nil && impExt != nil && impExt.GetGpId() != "" {
		return impExt.GetGpId()
	}

	if imp.TagID != "" {
		return imp.TagID
	}

	if impExt != nil {
		impExtData := impExt.GetData()
		if impExtData != nil && impExtData.PbAdslot != "" {
			return impExtData.PbAdslot
		}

		prebidExt := impExt.GetPrebid()
		if prebidExt != nil && prebidExt.StoredRequest != nil && prebidExt.StoredRequest.ID != "" {
			return prebidExt.StoredRequest.ID
		}
	}

	return adUnitCode
}

// isMobileDevice returns true if device is mobile
func isMobileDevice(userAgent string) bool {
	isMobile, err := regexp.MatchString("(?i)Phone|iPhone|Android.*Mobile|Mobile.*Android", userAgent)
	if err != nil {
		return false
	}
	return isMobile
}

// isTabletDevice returns true if device is tablet
func isTabletDevice(userAgent string) bool {
	isTablet, err := regexp.MatchString("(?i)tablet|iPad|touch.*Windows NT|Windows NT.*touch|Android", userAgent)
	if err != nil {
		return false
	}
	return isTablet
}

// prepareRuleCombinations prepares rule combinations based on schema dimensions and request fields
func prepareRuleCombinations(keys []string, delimiter string) []string {
	var schemaFields []string

	numSchemaFields := len(keys)
	ruleKey := newFloorRuleKeys(delimiter)
	for i := 0; i < numSchemaFields; i++ {
		schemaFields = append(schemaFields, strings.ToLower(keys[i]))
	}
	ruleKey.appendRuleKey(schemaFields)

	for numWildCard := 1; numWildCard <= numSchemaFields; numWildCard++ {
		newComb := generateCombinations(numSchemaFields, numWildCard)
		sortCombinations(newComb, numSchemaFields)

		for i := 0; i < len(newComb); i++ {
			eachSet := make([]string, numSchemaFields)
			copy(eachSet, schemaFields)
			for j := 0; j < len(newComb[i]); j++ {
				eachSet[newComb[i][j]] = catchAll
			}
			ruleKey.appendRuleKey(eachSet)
		}
	}
	return ruleKey.getAllRuleKeys()
}

// generateCombinations generates every permutation for the given number of fields with the specified number of
// wildcards. Permutations are returned as a list of integer lists where each integer list represents a single
// permutation with each integer indicating the position of the fields that are wildcards
// source: https://docs.prebid.org/dev-docs/modules/floors.html#rule-selection-process
func generateCombinations(numSchemaFields int, numWildCard int) (comb [][]int) {

	for subsetBits := 1; subsetBits < (1 << numSchemaFields); subsetBits++ {
		if bits.OnesCount(uint(subsetBits)) != numWildCard {
			continue
		}
		var subset []int
		for object := 0; object < numSchemaFields; object++ {
			if (subsetBits>>object)&1 == 1 {
				subset = append(subset, object)
			}
		}
		comb = append(comb, subset)
	}
	return comb
}

// sortCombinations sorts the list of combinations from most specific to least specific. A combination is considered more specific than
// another combination if it has more exact values (less wildcards). If two combinations have the same number of wildcards, a combination
// is considered more specific than another if its left-most fields are more exact.
func sortCombinations(comb [][]int, numSchemaFields int) {
	totalComb := 1 << numSchemaFields

	sort.SliceStable(comb, func(i, j int) bool {
		wt1 := 0
		for k := 0; k < len(comb[i]); k++ {
			wt1 += 1 << (totalComb - comb[i][k])
		}

		wt2 := 0
		for k := 0; k < len(comb[j]); k++ {
			wt2 += 1 << (totalComb - comb[j][k])
		}
		return wt1 < wt2
	})
}

// ruleKeys defines struct used for maintaining rule combinations generated from schema fields and reqeust values.
type ruleKeys struct {
	keyMap    map[string]bool
	keys      []string
	delimiter string
}

// newFloorRuleKeys allocates and initialise ruleKeys
func newFloorRuleKeys(delimiter string) *ruleKeys {
	rulekey := new(ruleKeys)
	rulekey.delimiter = delimiter
	rulekey.keyMap = map[string]bool{}
	return rulekey
}

// appendRuleKey appends unique rules keys into ruleKeys array
func (r *ruleKeys) appendRuleKey(rawKey []string) {
	var key string
	key = rawKey[0]
	for j := 1; j < len(rawKey); j++ {
		key += r.delimiter + rawKey[j]
	}

	if _, found := r.keyMap[key]; !found {
		r.keyMap[key] = true
		r.keys = append(r.keys, key)
	}
}

// getAllRuleKeys returns all the rules prepared
func (r *ruleKeys) getAllRuleKeys() []string {
	return r.keys
}
