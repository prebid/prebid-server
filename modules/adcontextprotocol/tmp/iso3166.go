package tmp

import "strings"

// normalizeCountryToAlpha2 converts a country code to its ISO 3166-1
// alpha-2 form for TMP wire emission. Accepts:
//
//   - 2-char input:   uppercased and returned as-is (already alpha-2)
//   - 3-char input:   looked up in iso3166Alpha3ToAlpha2 and returned as
//     alpha-2 when the code is recognized
//   - anything else:  returned "" so the caller omits the field
//
// The 3-char path exists because OpenRTB `device.geo.country` is defined
// as ISO 3166-1 alpha-3 (`"USA"`), whereas the TMP wire's `country` field
// is alpha-2 (`"US"`, per adcp-go's tmproto validator). A spec-compliant
// OpenRTB caller would otherwise produce a hard 400 at the verifier.
//
// The first-two-characters heuristic is deliberately NOT used — it maps
// `AUT` (Austria) to `AU` (Australia), `IRL` (Ireland) to `IR` (Iran),
// `PRT` (Portugal) to `PR` (Puerto Rico), and other silent
// misclassifications. Only the lookup table below is trusted.
func normalizeCountryToAlpha2(in string) string {
	s := strings.ToUpper(strings.TrimSpace(in))
	switch len(s) {
	case 0:
		return ""
	case 2:
		if isASCIILettersOnly(s) {
			return s
		}
		return ""
	case 3:
		if a2, ok := iso3166Alpha3ToAlpha2[s]; ok {
			return a2
		}
		return ""
	default:
		return ""
	}
}

func isASCIILettersOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// iso3166Alpha3ToAlpha2 maps every ISO 3166-1 alpha-3 code to its
// corresponding alpha-2. Sourced from the ISO 3166-1 register; this
// list changes rarely and is safe to embed inline.
var iso3166Alpha3ToAlpha2 = map[string]string{
	"AFG": "AF", "ALA": "AX", "ALB": "AL", "DZA": "DZ", "ASM": "AS",
	"AND": "AD", "AGO": "AO", "AIA": "AI", "ATA": "AQ", "ATG": "AG",
	"ARG": "AR", "ARM": "AM", "ABW": "AW", "AUS": "AU", "AUT": "AT",
	"AZE": "AZ", "BHS": "BS", "BHR": "BH", "BGD": "BD", "BRB": "BB",
	"BLR": "BY", "BEL": "BE", "BLZ": "BZ", "BEN": "BJ", "BMU": "BM",
	"BTN": "BT", "BOL": "BO", "BES": "BQ", "BIH": "BA", "BWA": "BW",
	"BVT": "BV", "BRA": "BR", "IOT": "IO", "BRN": "BN", "BGR": "BG",
	"BFA": "BF", "BDI": "BI", "CPV": "CV", "KHM": "KH", "CMR": "CM",
	"CAN": "CA", "CYM": "KY", "CAF": "CF", "TCD": "TD", "CHL": "CL",
	"CHN": "CN", "CXR": "CX", "CCK": "CC", "COL": "CO", "COM": "KM",
	"COG": "CG", "COD": "CD", "COK": "CK", "CRI": "CR", "CIV": "CI",
	"HRV": "HR", "CUB": "CU", "CUW": "CW", "CYP": "CY", "CZE": "CZ",
	"DNK": "DK", "DJI": "DJ", "DMA": "DM", "DOM": "DO", "ECU": "EC",
	"EGY": "EG", "SLV": "SV", "GNQ": "GQ", "ERI": "ER", "EST": "EE",
	"SWZ": "SZ", "ETH": "ET", "FLK": "FK", "FRO": "FO", "FJI": "FJ",
	"FIN": "FI", "FRA": "FR", "GUF": "GF", "PYF": "PF", "ATF": "TF",
	"GAB": "GA", "GMB": "GM", "GEO": "GE", "DEU": "DE", "GHA": "GH",
	"GIB": "GI", "GRC": "GR", "GRL": "GL", "GRD": "GD", "GLP": "GP",
	"GUM": "GU", "GTM": "GT", "GGY": "GG", "GIN": "GN", "GNB": "GW",
	"GUY": "GY", "HTI": "HT", "HMD": "HM", "VAT": "VA", "HND": "HN",
	"HKG": "HK", "HUN": "HU", "ISL": "IS", "IND": "IN", "IDN": "ID",
	"IRN": "IR", "IRQ": "IQ", "IRL": "IE", "IMN": "IM", "ISR": "IL",
	"ITA": "IT", "JAM": "JM", "JPN": "JP", "JEY": "JE", "JOR": "JO",
	"KAZ": "KZ", "KEN": "KE", "KIR": "KI", "PRK": "KP", "KOR": "KR",
	"KWT": "KW", "KGZ": "KG", "LAO": "LA", "LVA": "LV", "LBN": "LB",
	"LSO": "LS", "LBR": "LR", "LBY": "LY", "LIE": "LI", "LTU": "LT",
	"LUX": "LU", "MAC": "MO", "MDG": "MG", "MWI": "MW", "MYS": "MY",
	"MDV": "MV", "MLI": "ML", "MLT": "MT", "MHL": "MH", "MTQ": "MQ",
	"MRT": "MR", "MUS": "MU", "MYT": "YT", "MEX": "MX", "FSM": "FM",
	"MDA": "MD", "MCO": "MC", "MNG": "MN", "MNE": "ME", "MSR": "MS",
	"MAR": "MA", "MOZ": "MZ", "MMR": "MM", "NAM": "NA", "NRU": "NR",
	"NPL": "NP", "NLD": "NL", "NCL": "NC", "NZL": "NZ", "NIC": "NI",
	"NER": "NE", "NGA": "NG", "NIU": "NU", "NFK": "NF", "MKD": "MK",
	"MNP": "MP", "NOR": "NO", "OMN": "OM", "PAK": "PK", "PLW": "PW",
	"PSE": "PS", "PAN": "PA", "PNG": "PG", "PRY": "PY", "PER": "PE",
	"PHL": "PH", "PCN": "PN", "POL": "PL", "PRT": "PT", "PRI": "PR",
	"QAT": "QA", "REU": "RE", "ROU": "RO", "RUS": "RU", "RWA": "RW",
	"BLM": "BL", "SHN": "SH", "KNA": "KN", "LCA": "LC", "MAF": "MF",
	"SPM": "PM", "VCT": "VC", "WSM": "WS", "SMR": "SM", "STP": "ST",
	"SAU": "SA", "SEN": "SN", "SRB": "RS", "SYC": "SC", "SLE": "SL",
	"SGP": "SG", "SXM": "SX", "SVK": "SK", "SVN": "SI", "SLB": "SB",
	"SOM": "SO", "ZAF": "ZA", "SGS": "GS", "SSD": "SS", "ESP": "ES",
	"LKA": "LK", "SDN": "SD", "SUR": "SR", "SJM": "SJ", "SWE": "SE",
	"CHE": "CH", "SYR": "SY", "TWN": "TW", "TJK": "TJ", "TZA": "TZ",
	"THA": "TH", "TLS": "TL", "TGO": "TG", "TKL": "TK", "TON": "TO",
	"TTO": "TT", "TUN": "TN", "TUR": "TR", "TKM": "TM", "TCA": "TC",
	"TUV": "TV", "UGA": "UG", "UKR": "UA", "ARE": "AE", "GBR": "GB",
	"USA": "US", "UMI": "UM", "URY": "UY", "UZB": "UZ", "VUT": "VU",
	"VEN": "VE", "VNM": "VN", "VGB": "VG", "VIR": "VI", "WLF": "WF",
	"ESH": "EH", "YEM": "YE", "ZMB": "ZM", "ZWE": "ZW",
}
