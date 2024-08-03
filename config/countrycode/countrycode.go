package countrycode

import (
	"strings"
)

type CountryCode struct {
	map2To3 map[string]string
	map3To2 map[string]string
}

func New() *CountryCode {
	return &CountryCode{
		map2To3: make(map[string]string),
		map3To2: make(map[string]string),
	}
}

// Load loads country code mapping data
func (c *CountryCode) Load(data string) {
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

	c.map2To3 = toAlpha3
	c.map3To2 = toAlpha2
}

// ToAlpha3 converts country code alpha2 to alpha3
func (c *CountryCode) ToAlpha3(alpha2 string) string {
	return c.map2To3[alpha2]
}

// ToAlpha2 converts country code alpha3 to alpha2
func (c *CountryCode) ToAlpha2(alpha3 string) string {
	return c.map3To2[alpha3]
}

var defaultCountryCode = New()

func Load(data string) {
	defaultCountryCode.Load(data)
}

func ToAlpha3(alpha2 string) string {
	return defaultCountryCode.ToAlpha3(alpha2)
}

func ToAlpha2(alpha3 string) string {
	return defaultCountryCode.ToAlpha2(alpha3)
}
