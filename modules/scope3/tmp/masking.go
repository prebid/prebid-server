package tmp

import (
	"encoding/json"
	"math"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// countryAlpha3ToAlpha2 converts an ISO 3166-1 alpha-3 code to alpha-2.
// Returns "" for unknown or empty input. Input is case-sensitive uppercase.
func countryAlpha3ToAlpha2(alpha3 string) string {
	return iso3166Alpha3ToAlpha2[alpha3]
}

// iso3166Alpha3ToAlpha2 is the static mapping of ISO 3166-1 alpha-3 codes
// (used by OpenRTB device.geo.country) to alpha-2 codes (required by TMP).
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
	"COD": "CD", "COG": "CG", "COK": "CK", "CRI": "CR", "CIV": "CI",
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
	"LUX": "LU", "MAC": "MO", "MKD": "MK", "MDG": "MG", "MWI": "MW",
	"MYS": "MY", "MDV": "MV", "MLI": "ML", "MLT": "MT", "MHL": "MH",
	"MTQ": "MQ", "MRT": "MR", "MUS": "MU", "MYT": "YT", "MEX": "MX",
	"FSM": "FM", "MDA": "MD", "MCO": "MC", "MNG": "MN", "MNE": "ME",
	"MSR": "MS", "MAR": "MA", "MOZ": "MZ", "MMR": "MM", "NAM": "NA",
	"NRU": "NR", "NPL": "NP", "NLD": "NL", "NCL": "NC", "NZL": "NZ",
	"NIC": "NI", "NER": "NE", "NGA": "NG", "NIU": "NU", "NFK": "NF",
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

// extractIdentities picks up to 3 identity tokens from the user object, in the
// order specified by preserveEids. Falls back to publisher user.id only when
// no eids match and user.id is non-empty.
//
// The spec hard-caps Identities at 3 entries (maxItems: 3) because of the TMPX
// HPKE plaintext byte budget. Builder validation already rejects preserveEids
// longer than 3, so this function trusts that bound.
func extractIdentities(user *openrtb2.User, preserveEids []string) []IdentityToken {
	if user == nil {
		return nil
	}

	var ext struct {
		EIDs []openrtb2.EID `json:"eids"`
	}
	if len(user.Ext) > 0 {
		_ = json.Unmarshal(user.Ext, &ext) // best effort; treat parse failure as no EIDs
	}

	bySource := make(map[string]string, len(ext.EIDs))
	for _, eid := range ext.EIDs {
		if len(eid.UIDs) == 0 {
			continue
		}
		if _, dup := bySource[eid.Source]; dup {
			continue
		}
		bySource[eid.Source] = eid.UIDs[0].ID
	}

	out := make([]IdentityToken, 0, len(preserveEids))
	for _, source := range preserveEids {
		if id, ok := bySource[source]; ok && id != "" {
			out = append(out, IdentityToken{UIDType: source, UserToken: id})
		}
	}

	if len(out) == 0 && user.ID != "" {
		out = append(out, IdentityToken{UIDType: "publisher_user_id", UserToken: user.ID})
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// maskBidRequest returns a deep copy with sensitive fields removed per config.
// Returns the original (unmodified) if masking is disabled.
func maskBidRequest(orig *openrtb2.BidRequest, cfg MaskingConfig) *openrtb2.BidRequest {
	if !cfg.Enabled {
		return orig
	}
	raw, err := json.Marshal(orig)
	if err != nil {
		return nil
	}
	var copyBR openrtb2.BidRequest
	if err := json.Unmarshal(raw, &copyBR); err != nil {
		return nil
	}
	maskUser(&copyBR, cfg.User)
	maskDevice(&copyBR, cfg.Device)
	maskGeo(&copyBR, cfg.Geo)
	return &copyBR
}

func maskUser(br *openrtb2.BidRequest, cfg UserMaskingConfig) {
	if br.User == nil {
		return
	}
	br.User.ID = ""
	br.User.BuyerUID = ""
	br.User.Yob = 0
	br.User.Gender = ""
	br.User.Data = nil
	br.User.Keywords = ""
	br.User.EIDs = filterEIDs(br.User.EIDs, cfg.PreserveEids)
}

func filterEIDs(eids []openrtb2.EID, allow []string) []openrtb2.EID {
	if len(allow) == 0 {
		return nil
	}
	allowSet := make(map[string]struct{}, len(allow))
	for _, a := range allow {
		allowSet[a] = struct{}{}
	}
	out := make([]openrtb2.EID, 0, len(eids))
	for _, e := range eids {
		if _, ok := allowSet[e.Source]; ok {
			out = append(out, e)
		}
	}
	return out
}

func maskDevice(br *openrtb2.BidRequest, cfg DeviceMaskingConfig) {
	if br.Device == nil {
		return
	}
	br.Device.IP = ""
	br.Device.IPv6 = ""
	if !cfg.PreserveMobileIds {
		br.Device.IFA = ""
		br.Device.DPIDMD5 = ""
		br.Device.DPIDSHA1 = ""
		br.Device.MACMD5 = ""
		br.Device.MACSHA1 = ""
		br.Device.DIDMD5 = ""
		br.Device.DIDSHA1 = ""
	}
}

func maskGeo(br *openrtb2.BidRequest, cfg GeoMaskingConfig) {
	if br.Device == nil || br.Device.Geo == nil {
		return
	}
	g := br.Device.Geo
	if !cfg.PreserveMetro {
		g.Metro = ""
	}
	if !cfg.PreserveZip {
		g.ZIP = ""
	}
	if !cfg.PreserveCity {
		g.City = ""
	}
	g.Accuracy = 0
	if g.Lat != nil {
		*g.Lat = truncateLatLong(*g.Lat, cfg.LatLongPrecision)
	}
	if g.Lon != nil {
		*g.Lon = truncateLatLong(*g.Lon, cfg.LatLongPrecision)
	}
}

func truncateLatLong(v float64, precision int) float64 {
	if precision <= 0 {
		return 0
	}
	mult := math.Pow(10, float64(precision))
	return math.Round(v*mult) / mult
}
