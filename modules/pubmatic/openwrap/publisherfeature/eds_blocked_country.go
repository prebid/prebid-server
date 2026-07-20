package publisherfeature

import (
	"strings"

	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

type edsBlockedCountries struct {
	countries [2]models.HashSet
	index     int
}

func newEDSBlockedCountries() edsBlockedCountries {
	return edsBlockedCountries{
		countries: [2]models.HashSet{
			make(models.HashSet),
			make(models.HashSet),
		},
		index: 0,
	}
}

// updateEDSBlockedCountries loads platform-level blocked countries from pubID=0 in publisherFeatureMap.
func (fe *feature) updateEDSBlockedCountries() {
	if fe.publisherFeature == nil {
		return
	}

	blockedCountries := make(models.HashSet)
	feature, ok := fe.publisherFeature[0]
	if ok {
		if val, ok := feature[models.FeatureEDSBlockedCountries]; ok && val.Enabled == 1 {
			for _, country := range strings.Split(val.Value, ",") {
				country = strings.TrimSpace(country)
				if country != "" {
					blockedCountries[country] = struct{}{}
				}
			}
		}
	}

	fe.edsBlockedCountries.countries[fe.edsBlockedCountries.index^1] = blockedCountries
	fe.edsBlockedCountries.index ^= 1
}

// IsEDSBlockedCountry returns true if the country is in the platform blocked list.
// When value is "*", all countries (including unknown) are blocked.
func (fe *feature) IsEDSBlockedCountry(countryCode string) bool {
	countries := fe.edsBlockedCountries.countries[fe.edsBlockedCountries.index]
	if _, blockAll := countries[models.EDSBlockedCountriesAll]; blockAll {
		return true
	}
	if countryCode == "" {
		return false
	}
	_, blocked := countries[countryCode]
	return blocked
}
