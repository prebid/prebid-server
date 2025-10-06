package rulesengine

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/rules"
)

type CountryExclusions = map[string][]string
type CountryGroups = map[string][]string

// defaultCountryGroups maps country group names to their member countries
var defaultCountryGroups = CountryGroups{
	"EEA": []string{
		"AUT", "BEL", "BGR", "CYP", "CZE", "DEU", "DNK", "EST", "ESP",
		"FIN", "FRA", "GRC", "HRV", "HUN", "IRL", "ISL", "ITA", "LIE",
		"LTU", "LUX", "LVA", "MLT", "NLD", "NOR", "POL", "PRT", "ROU",
		"SWE", "SVN", "SVK",
	},
}

// parsedConfig holds the parsed geoscope configuration for a bidder
type parsedConfig struct {
	bidder                  string
	globalIncluded          bool
	countryGroupsIncluded   []string
	countryGroupsExcluded   []string
	singleCountriesIncluded []string
	singleCountriesExcluded []string
}

// NewBidderConfigRuleSetBuilder creates a new instance of bidderConfigRuleSetBuilder
func NewBidderConfigRuleSetBuilder[T1 RequestWrapper, T2 ProcessedAuctionHookResult](
	geoscopeConfig map[string][]string, setDefinitions map[string][]string,
) *bidderConfigRuleSetBuilder[T1, T2] {

	return &bidderConfigRuleSetBuilder[T1, T2]{
		countryGroups:  mergeCountryGroups(defaultCountryGroups, setDefinitions),
		geoscopeConfig: geoscopeConfig,
	}
}

// mergeCountryGroups merges the default country groups with any account-defined set definitions
func mergeCountryGroups(defaults, setDefinitions CountryGroups) CountryGroups {
	countryGroups := make(CountryGroups, len(defaults))
	for k, v := range defaults {
		copied := make([]string, len(v))
		copy(copied, v)
		countryGroups[strings.ToUpper(k)] = copied
	}
	if setDefinitions != nil {
		for k, v := range setDefinitions {
			copied := make([]string, len(v))
			copy(copied, v)
			countryGroups[strings.ToUpper(k)] = copied
		}
	}
	return countryGroups
}

// bidderConfigRuleSetBuilder builds a dynamic rule set based on the geoscope annotations in the
// static bidder-info bidder YAML files
type bidderConfigRuleSetBuilder[T1 RequestWrapper, T2 ProcessedAuctionHookResult] struct {
	countryGroups  CountryGroups
	geoscopeConfig map[string][]string
}

// Build constructs the dynamic rule set
func (b *bidderConfigRuleSetBuilder[T1, T2]) Build(tree *rules.Tree[RequestWrapper, ProcessedAuctionHookResult]) error {
	if len(b.geoscopeConfig) == 0 || tree.Root == nil {
		return nil
	}

	schemaFunc, err := rules.NewDeviceCountry(json.RawMessage(`{}`))
	if err != nil {
		return err
	}
	tree.Root.SchemaFunction = schemaFunc

	countryExclusions := b.initializeCountryExclusions()

	for bidder, geoscopes := range b.geoscopeConfig {
		parsedCfg, err := b.parseBidderGeoscopes(bidder, geoscopes)
		if err != nil {
			return err
		}

		if parsedCfg.globalIncluded {
			continue
		} else if len(parsedCfg.countryGroupsIncluded) > 0 {
			b.markAllowListWithGroupsExclusions(parsedCfg, countryExclusions)
		} else if len(parsedCfg.countryGroupsExcluded) > 0 {
			b.markBlockListWithGroupsExclusions(parsedCfg, countryExclusions)
		} else if len(parsedCfg.singleCountriesIncluded) > 0 {
			b.markAllowListExclusions(parsedCfg, countryExclusions)
		} else if len(parsedCfg.singleCountriesExcluded) > 0 {
			b.markBlockListExclusions(parsedCfg, countryExclusions)
		}
	}

	addTreeNodes(tree.Root, countryExclusions)

	return nil
}

// populateCountries creates a exclusions map with its set of keys being all geoscope annotation countries explicitly
// called out along with all countries in a geoscope annotation country group
// The bidder exclusions for each country is empty
func (b *bidderConfigRuleSetBuilder[T1, T2]) initializeCountryExclusions() CountryExclusions {

	countryExclusions := map[string][]string{}
	countryGroupsSeen := map[string]struct{}{}

	for _, geoscopes := range b.geoscopeConfig {
		for _, geoscope := range geoscopes {
			if geoscope == "GLOBAL" {
				continue
			}
			if strings.HasPrefix(geoscope, "!") {
				geoscope = strings.TrimPrefix(geoscope, "!")
			}
			if countryGroup, groupFound := b.countryGroups[geoscope]; groupFound {
				if _, countrySeen := countryGroupsSeen[geoscope]; !countrySeen {
					for _, country := range countryGroup {
						if _, exists := countryExclusions[country]; !exists {
							countryExclusions[country] = []string{}
						}
					}
					countryGroupsSeen[geoscope] = struct{}{}
				}
			} else if _, exists := countryExclusions[geoscope]; !exists {
				countryExclusions[geoscope] = []string{}
			}
		}
	}
	countryExclusions["*"] = []string{}

	return countryExclusions
}

// parseBidderGeoscopes parses the geoscope directives for a bidder and returns a parsedConfig struct
func (b *bidderConfigRuleSetBuilder[T1, T2]) parseBidderGeoscopes(bidder string, geoscopes []string) (parsedConfig, error) {
	cfg := parsedConfig{bidder: bidder}

	for _, geoscope := range geoscopes {
		if geoscope == "GLOBAL" {
			cfg.globalIncluded = true
		} else if b.includedCountryGroup(geoscope) {
			cfg.countryGroupsIncluded = append(cfg.countryGroupsIncluded, geoscope)
		} else if b.excludedCountryGroup(geoscope) {
			cfg.countryGroupsExcluded = append(cfg.countryGroupsExcluded, strings.TrimPrefix(geoscope, "!"))
		} else if b.includedSingleCountry(geoscope) {
			cfg.singleCountriesIncluded = append(cfg.singleCountriesIncluded, geoscope)
		} else if b.excludedSingleCountry(geoscope) {
			cfg.singleCountriesExcluded = append(cfg.singleCountriesExcluded, strings.TrimPrefix(geoscope, "!"))
		} else {
			return cfg, errors.New("unknown geoscope type: " + geoscope)
		}
	}

	return cfg, nil
}

// markAllowListWithGroupsExclusions analyzes the parsed bidder config to determine which countries should be excluded.
// It is called when it has been determined that the bidder geoscope information should be interpreted as
// an allow list and there is at least one allowed country group specified in the geoscope directives for
// the bidder.
func (b *bidderConfigRuleSetBuilder[T1, T2]) markAllowListWithGroupsExclusions(cfg parsedConfig, countryExclusions CountryExclusions) {
	for country, excludedBidders := range countryExclusions {
		inGroup := b.countryInAGroup(country, cfg.countryGroupsIncluded)
		specificallyIncluded := slices.Contains(cfg.singleCountriesIncluded, country)
		specificallyExcluded := slices.Contains(cfg.singleCountriesExcluded, country)

		if inGroup && specificallyExcluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		} else if !inGroup && !specificallyIncluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		}
	}
}

// markBlockListWithGroupsExclusions analyzes the parsed bidder config to determine which countries should be excluded.
// It is called when it has been determined that the bidder geoscope information should be interpreted as
// a block list and there is at least one blocked country group specified in the geoscope directives for
// the bidder.
func (b *bidderConfigRuleSetBuilder[T1, T2]) markBlockListWithGroupsExclusions(cfg parsedConfig, countryExclusions CountryExclusions) {
	for country, excludedBidders := range countryExclusions {
		inGroup := b.countryInAGroup(country, cfg.countryGroupsExcluded)
		specificallyIncluded := slices.Contains(cfg.singleCountriesIncluded, country)
		specificallyExcluded := slices.Contains(cfg.singleCountriesExcluded, country)

		if inGroup && !specificallyIncluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		} else if !inGroup && specificallyExcluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		}
	}
}

// markAllowListExclusions analyzes the parsed bidder config to determine which countries should be excluded.
// It is called when it has been determined that the bidder geoscope information should be interpreted as
// an allow list and there are no country groups specified in the geoscope directives for the bidder.
func (b *bidderConfigRuleSetBuilder[T1, T2]) markAllowListExclusions(cfg parsedConfig, countryExclusions CountryExclusions) {
	for country, excludedBidders := range countryExclusions {
		specificallyIncluded := slices.Contains(cfg.singleCountriesIncluded, country)

		if !specificallyIncluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		}
	}
}

// markBlockListExclusions analyzes the parsed bidder config to determine which countries should be excluded.
// It is called when it has been determined that the bidder geoscope information should be interpreted as
// a block list and there are no country groups specified in the geoscope directives for the bidder.
func (b *bidderConfigRuleSetBuilder[T1, T2]) markBlockListExclusions(cfg parsedConfig, countryExclusions CountryExclusions) {
	for country, excludedBidders := range countryExclusions {
		specificallyExcluded := slices.Contains(cfg.singleCountriesExcluded, country)

		if specificallyExcluded {
			excludedBidders = append(excludedBidders, cfg.bidder)
			countryExclusions[country] = excludedBidders
		}
	}
}

// includeCountryGroup checks if the geoscope annotation refers to an included country group
func (b *bidderConfigRuleSetBuilder[T1, T2]) includedCountryGroup(geoscope string) bool {
	return b.countryGroups[geoscope] != nil
}

// excludedCountryGroup checks if the geoscope annotation refers to an excluded country group
func (b *bidderConfigRuleSetBuilder[T1, T2]) excludedCountryGroup(geoscope string) bool {
	return strings.HasPrefix(geoscope, "!") && b.countryGroups[geoscope[1:]] != nil
}

// includedSingleCountry checks if the geoscope annotation refers to an included country
func (b *bidderConfigRuleSetBuilder[T1, T2]) includedSingleCountry(geoscope string) bool {
	return !strings.HasPrefix(geoscope, "!") && b.countryGroups[geoscope] == nil
}

// excludedSingleCountry checks if the geoscope annotation refers to an excluded country
func (b *bidderConfigRuleSetBuilder[T1, T2]) excludedSingleCountry(geoscope string) bool {
	return strings.HasPrefix(geoscope, "!") && b.countryGroups[geoscope] == nil
}

// countryInAGroup checks if the given country is part of any of the specified country groups
func (b *bidderConfigRuleSetBuilder[T1, T2]) countryInAGroup(country string, groups []string) bool {
	for _, groupName := range groups {
		if countryList, exists := b.countryGroups[groupName]; exists {
			for _, countryInGroup := range countryList {
				if country == countryInGroup {
					return true
				}
			}
		}
	}
	return false
}

// addTreeNodes adds the country exclusion nodes to the rules tree
func addTreeNodes(root *rules.Node[RequestWrapper, ProcessedAuctionHookResult], countryExclusions CountryExclusions) {
	root.Children = make(map[string]*rules.Node[RequestWrapper, ProcessedAuctionHookResult])

	for country, excludedBidders := range countryExclusions {

		node := rules.Node[RequestWrapper, ProcessedAuctionHookResult]{
			SchemaFunction: nil,
			ResultFunctions: []rules.ResultFunction[RequestWrapper, ProcessedAuctionHookResult]{
				&ExcludeBidders{
					Args: config.ResultFuncParams{
						Bidders: excludedBidders,
					},
				},
			},
			Children: nil,
		}

		root.Children[country] = &node
	}
}
