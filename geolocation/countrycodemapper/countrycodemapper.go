package countrycodemapper

import (
	"strings"
)

type CountryCodeMapper struct {
	alpha2ToAlpha3CountryCodes map[string]string
	alpha3ToAlpha2CountryCodes map[string]string
}

func New() *CountryCodeMapper {
	return &CountryCodeMapper{
		alpha2ToAlpha3CountryCodes: make(map[string]string),
		alpha3ToAlpha2CountryCodes: make(map[string]string),
	}
}

// Load loads country code mapping data
func (c *CountryCodeMapper) Load(data string) {
	toAlpha2 := make(map[string]string)
	toAlpha3 := make(map[string]string)
	for _, line := range strings.Split(data, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		alpha2 := strings.TrimSpace(fields[0])
		alpha3 := strings.TrimSpace(fields[1])
		toAlpha2[alpha3] = alpha2
		toAlpha3[alpha2] = alpha3
	}

	c.alpha2ToAlpha3CountryCodes = toAlpha3
	c.alpha3ToAlpha2CountryCodes = toAlpha2
}

// MapToAlpha3 converts country code alpha2 to alpha3
func (c *CountryCodeMapper) MapToAlpha3(alpha2 string) string {
	return c.alpha2ToAlpha3CountryCodes[alpha2]
}

// MapToAlpha2 converts country code alpha3 to alpha2
func (c *CountryCodeMapper) MapToAlpha2(alpha3 string) string {
	return c.alpha3ToAlpha2CountryCodes[alpha3]
}

var defaultCountryCodeMapper = New()

// Load loads country code mapping data into the default mapper
// This is NOT thread-safe, so it should be called before any concurrent access
func Load(data string) {
	defaultCountryCodeMapper.Load(data)
}

func MapToAlpha3(alpha2 string) string {
	return defaultCountryCodeMapper.MapToAlpha3(alpha2)
}

func MapToAlpha2(alpha3 string) string {
	return defaultCountryCodeMapper.MapToAlpha2(alpha3)
}
