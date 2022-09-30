package config

import "github.com/spf13/viper"

/*

Better Move SetupViper() -> SetDefault() patches to pbs.yaml (app-resources.yaml) instead of setupViper()

metrics:
	disabled_metrics:
		account_stored_responses: false


http_client:
	tls_handshake_timeout: 0
	response_header_timeout: 0
	dial_timeout: 0
	dial_keepalive: 0

category_mapping:
	filesystem:
		directorypath: "/home/http/GO_SERVER/dmhbserver/static/category-mapping"

adapters:
	ix:
		disabled: false
		endpoint: "http://exchange.indexww.com/pbs?p=192919"
	pangle:
		disabled: false
		endpoint: "https://api16-access-sg.pangle.io/api/ad/union/openrtb/get_ads/"
	rubicon:
		disabled: false
	spotx:
		endpoint: "https://search.spotxchange.com/openrtb/2.3/dados"
	vastbidder:
		endpoint: "https://test.com"
	vrtcal:
		endpoint: "http://rtb.vrtcal.com/bidder_prebid.vap?ssp=1812"

gdpr:
	default_value: 0
	usersync_if_ambiguous: true

*/

func setupViperOW(v *viper.Viper) {
	v.SetDefault("http_client.tls_handshake_timeout", 0)   //no timeout
	v.SetDefault("http_client.response_header_timeout", 0) //unlimited
	v.SetDefault("http_client.dial_timeout", 0)            //no timeout
	v.SetDefault("http_client.dial_keepalive", 0)          //no restriction
	v.SetDefault("category_mapping.filesystem.directorypath", "/home/http/GO_SERVER/dmhbserver/static/category-mapping")
	v.SetDefault("adapters.ix.disabled", false)
	v.SetDefault("adapters.ix.endpoint", "http://exchange.indexww.com/pbs?p=192919")
	v.SetDefault("adapters.pangle.disabled", false)
	v.SetDefault("adapters.pangle.endpoint", "https://api16-access-sg.pangle.io/api/ad/union/openrtb/get_ads/")
	v.SetDefault("adapters.rubicon.disabled", false)
	v.SetDefault("adapters.spotx.endpoint", "https://search.spotxchange.com/openrtb/2.3/dados")
	v.SetDefault("adapters.vastbidder.endpoint", "https://test.com")
	v.SetDefault("adapters.vrtcal.endpoint", "http://rtb.vrtcal.com/bidder_prebid.vap?ssp=1812")
	v.SetDefault("adapters.yahoossp.disabled", true)
	v.SetDefault("gdpr.default_value", "0")
	v.SetDefault("gdpr.usersync_if_ambiguous", true)
}
