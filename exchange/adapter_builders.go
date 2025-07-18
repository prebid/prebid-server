package exchange

import (
	"github.com/prebid/prebid-server/v3/adapters"
	ttx "github.com/prebid/prebid-server/v3/adapters/33across"
	"github.com/prebid/prebid-server/v3/adapters/aax"
	"github.com/prebid/prebid-server/v3/adapters/aceex"
	"github.com/prebid/prebid-server/v3/adapters/acuityads"
	"github.com/prebid/prebid-server/v3/adapters/adagio"
	"github.com/prebid/prebid-server/v3/adapters/adelement"
	"github.com/prebid/prebid-server/v3/adapters/adf"
	"github.com/prebid/prebid-server/v3/adapters/adgeneration"
	"github.com/prebid/prebid-server/v3/adapters/adhese"
	"github.com/prebid/prebid-server/v3/adapters/adkernel"
	"github.com/prebid/prebid-server/v3/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/v3/adapters/adman"
	"github.com/prebid/prebid-server/v3/adapters/admatic"
	"github.com/prebid/prebid-server/v3/adapters/admixer"
	"github.com/prebid/prebid-server/v3/adapters/adnuntius"
	"github.com/prebid/prebid-server/v3/adapters/adocean"
	"github.com/prebid/prebid-server/v3/adapters/adoppler"
	"github.com/prebid/prebid-server/v3/adapters/adot"
	"github.com/prebid/prebid-server/v3/adapters/adpone"
	"github.com/prebid/prebid-server/v3/adapters/adprime"
	"github.com/prebid/prebid-server/v3/adapters/adquery"
	"github.com/prebid/prebid-server/v3/adapters/adrino"
	"github.com/prebid/prebid-server/v3/adapters/ads_interactive"
	"github.com/prebid/prebid-server/v3/adapters/adsinteractive"
	"github.com/prebid/prebid-server/v3/adapters/adtarget"
	"github.com/prebid/prebid-server/v3/adapters/adtelligent"
	"github.com/prebid/prebid-server/v3/adapters/adtonos"
	"github.com/prebid/prebid-server/v3/adapters/adtrgtme"
	"github.com/prebid/prebid-server/v3/adapters/aduptech"
	"github.com/prebid/prebid-server/v3/adapters/advangelists"
	"github.com/prebid/prebid-server/v3/adapters/adverxo"
	"github.com/prebid/prebid-server/v3/adapters/adview"
	"github.com/prebid/prebid-server/v3/adapters/adxcg"
	"github.com/prebid/prebid-server/v3/adapters/adyoulike"
	"github.com/prebid/prebid-server/v3/adapters/aidem"
	"github.com/prebid/prebid-server/v3/adapters/aja"
	"github.com/prebid/prebid-server/v3/adapters/akcelo"
	"github.com/prebid/prebid-server/v3/adapters/algorix"
	"github.com/prebid/prebid-server/v3/adapters/alkimi"
	"github.com/prebid/prebid-server/v3/adapters/amx"
	"github.com/prebid/prebid-server/v3/adapters/apacdex"
	"github.com/prebid/prebid-server/v3/adapters/appnexus"
	"github.com/prebid/prebid-server/v3/adapters/appush"
	"github.com/prebid/prebid-server/v3/adapters/aso"
	"github.com/prebid/prebid-server/v3/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/v3/adapters/automatad"
	"github.com/prebid/prebid-server/v3/adapters/avocet"
	"github.com/prebid/prebid-server/v3/adapters/axis"
	"github.com/prebid/prebid-server/v3/adapters/axonix"
	"github.com/prebid/prebid-server/v3/adapters/beachfront"
	"github.com/prebid/prebid-server/v3/adapters/beintoo"
	"github.com/prebid/prebid-server/v3/adapters/bematterfull"
	"github.com/prebid/prebid-server/v3/adapters/between"
	"github.com/prebid/prebid-server/v3/adapters/beyondmedia"
	"github.com/prebid/prebid-server/v3/adapters/bidmachine"
	"github.com/prebid/prebid-server/v3/adapters/bidmatic"
	"github.com/prebid/prebid-server/v3/adapters/bidmyadz"
	"github.com/prebid/prebid-server/v3/adapters/bidscube"
	"github.com/prebid/prebid-server/v3/adapters/bidstack"
	"github.com/prebid/prebid-server/v3/adapters/bidtheatre"
	"github.com/prebid/prebid-server/v3/adapters/bigoad"
	"github.com/prebid/prebid-server/v3/adapters/blasto"
	"github.com/prebid/prebid-server/v3/adapters/bliink"
	"github.com/prebid/prebid-server/v3/adapters/blue"
	"github.com/prebid/prebid-server/v3/adapters/bluesea"
	"github.com/prebid/prebid-server/v3/adapters/bmtm"
	"github.com/prebid/prebid-server/v3/adapters/boldwin"
	"github.com/prebid/prebid-server/v3/adapters/brave"
	"github.com/prebid/prebid-server/v3/adapters/bwx"
	cadentaperturemx "github.com/prebid/prebid-server/v3/adapters/cadent_aperture_mx"
	"github.com/prebid/prebid-server/v3/adapters/ccx"
	"github.com/prebid/prebid-server/v3/adapters/cointraffic"
	"github.com/prebid/prebid-server/v3/adapters/coinzilla"
	"github.com/prebid/prebid-server/v3/adapters/colossus"
	"github.com/prebid/prebid-server/v3/adapters/compass"
	"github.com/prebid/prebid-server/v3/adapters/concert"
	"github.com/prebid/prebid-server/v3/adapters/connatix"
	"github.com/prebid/prebid-server/v3/adapters/connectad"
	"github.com/prebid/prebid-server/v3/adapters/consumable"
	"github.com/prebid/prebid-server/v3/adapters/conversant"
	"github.com/prebid/prebid-server/v3/adapters/copper6ssp"
	"github.com/prebid/prebid-server/v3/adapters/cpmstar"
	"github.com/prebid/prebid-server/v3/adapters/criteo"
	"github.com/prebid/prebid-server/v3/adapters/cwire"
	"github.com/prebid/prebid-server/v3/adapters/datablocks"
	"github.com/prebid/prebid-server/v3/adapters/decenterads"
	"github.com/prebid/prebid-server/v3/adapters/deepintent"
	"github.com/prebid/prebid-server/v3/adapters/definemedia"
	"github.com/prebid/prebid-server/v3/adapters/dianomi"
	"github.com/prebid/prebid-server/v3/adapters/displayio"
	"github.com/prebid/prebid-server/v3/adapters/dmx"
	"github.com/prebid/prebid-server/v3/adapters/driftpixel"
	"github.com/prebid/prebid-server/v3/adapters/dxkulture"
	evolution "github.com/prebid/prebid-server/v3/adapters/e_volution"
	"github.com/prebid/prebid-server/v3/adapters/edge226"
	"github.com/prebid/prebid-server/v3/adapters/emtv"
	"github.com/prebid/prebid-server/v3/adapters/eplanning"
	"github.com/prebid/prebid-server/v3/adapters/epom"
	"github.com/prebid/prebid-server/v3/adapters/escalax"
	"github.com/prebid/prebid-server/v3/adapters/exco"
	"github.com/prebid/prebid-server/v3/adapters/feedad"
	"github.com/prebid/prebid-server/v3/adapters/flatads"
	"github.com/prebid/prebid-server/v3/adapters/flipp"
	"github.com/prebid/prebid-server/v3/adapters/freewheelssp"
	"github.com/prebid/prebid-server/v3/adapters/frvradn"
	"github.com/prebid/prebid-server/v3/adapters/fwssp"
	"github.com/prebid/prebid-server/v3/adapters/gamma"
	"github.com/prebid/prebid-server/v3/adapters/gamoshi"
	"github.com/prebid/prebid-server/v3/adapters/globalsun"
	"github.com/prebid/prebid-server/v3/adapters/gothamads"
	"github.com/prebid/prebid-server/v3/adapters/grid"
	"github.com/prebid/prebid-server/v3/adapters/gumgum"
	"github.com/prebid/prebid-server/v3/adapters/huaweiads"
	"github.com/prebid/prebid-server/v3/adapters/imds"
	"github.com/prebid/prebid-server/v3/adapters/impactify"
	"github.com/prebid/prebid-server/v3/adapters/improvedigital"
	"github.com/prebid/prebid-server/v3/adapters/infytv"
	"github.com/prebid/prebid-server/v3/adapters/inmobi"
	"github.com/prebid/prebid-server/v3/adapters/insticator"
	"github.com/prebid/prebid-server/v3/adapters/interactiveoffers"
	"github.com/prebid/prebid-server/v3/adapters/intertech"
	"github.com/prebid/prebid-server/v3/adapters/invibes"
	"github.com/prebid/prebid-server/v3/adapters/iqx"
	"github.com/prebid/prebid-server/v3/adapters/iqzone"
	"github.com/prebid/prebid-server/v3/adapters/ix"
	"github.com/prebid/prebid-server/v3/adapters/jixie"
	"github.com/prebid/prebid-server/v3/adapters/kargo"
	"github.com/prebid/prebid-server/v3/adapters/kayzen"
	"github.com/prebid/prebid-server/v3/adapters/kidoz"
	"github.com/prebid/prebid-server/v3/adapters/kiviads"
	"github.com/prebid/prebid-server/v3/adapters/kobler"
	"github.com/prebid/prebid-server/v3/adapters/krushmedia"
	"github.com/prebid/prebid-server/v3/adapters/kueezrtb"
	"github.com/prebid/prebid-server/v3/adapters/lemmadigital"
	"github.com/prebid/prebid-server/v3/adapters/limelightDigital"
	lmkiviads "github.com/prebid/prebid-server/v3/adapters/lm_kiviads"
	"github.com/prebid/prebid-server/v3/adapters/lockerdome"
	"github.com/prebid/prebid-server/v3/adapters/logan"
	"github.com/prebid/prebid-server/v3/adapters/logicad"
	"github.com/prebid/prebid-server/v3/adapters/loopme"
	"github.com/prebid/prebid-server/v3/adapters/loyal"
	"github.com/prebid/prebid-server/v3/adapters/lunamedia"
	"github.com/prebid/prebid-server/v3/adapters/mabidder"
	"github.com/prebid/prebid-server/v3/adapters/madsense"
	"github.com/prebid/prebid-server/v3/adapters/madvertise"
	"github.com/prebid/prebid-server/v3/adapters/marsmedia"
	"github.com/prebid/prebid-server/v3/adapters/mediago"
	"github.com/prebid/prebid-server/v3/adapters/medianet"
	"github.com/prebid/prebid-server/v3/adapters/mediasquare"
	"github.com/prebid/prebid-server/v3/adapters/melozen"
	"github.com/prebid/prebid-server/v3/adapters/metax"
	"github.com/prebid/prebid-server/v3/adapters/mgid"
	"github.com/prebid/prebid-server/v3/adapters/mgidX"
	"github.com/prebid/prebid-server/v3/adapters/minutemedia"
	"github.com/prebid/prebid-server/v3/adapters/missena"
	"github.com/prebid/prebid-server/v3/adapters/mobfoxpb"
	"github.com/prebid/prebid-server/v3/adapters/mobilefuse"
	"github.com/prebid/prebid-server/v3/adapters/mobkoi"
	"github.com/prebid/prebid-server/v3/adapters/motorik"
	"github.com/prebid/prebid-server/v3/adapters/nativo"
	"github.com/prebid/prebid-server/v3/adapters/nextmillennium"
	"github.com/prebid/prebid-server/v3/adapters/nexx360"
	"github.com/prebid/prebid-server/v3/adapters/nobid"
	"github.com/prebid/prebid-server/v3/adapters/ogury"
	"github.com/prebid/prebid-server/v3/adapters/oms"
	"github.com/prebid/prebid-server/v3/adapters/onetag"
	"github.com/prebid/prebid-server/v3/adapters/openweb"
	"github.com/prebid/prebid-server/v3/adapters/openx"
	"github.com/prebid/prebid-server/v3/adapters/operaads"
	"github.com/prebid/prebid-server/v3/adapters/optidigital"
	"github.com/prebid/prebid-server/v3/adapters/oraki"
	"github.com/prebid/prebid-server/v3/adapters/orbidder"
	"github.com/prebid/prebid-server/v3/adapters/outbrain"
	"github.com/prebid/prebid-server/v3/adapters/ownadx"
	"github.com/prebid/prebid-server/v3/adapters/pangle"
	"github.com/prebid/prebid-server/v3/adapters/pgamssp"
	"github.com/prebid/prebid-server/v3/adapters/playdigo"
	"github.com/prebid/prebid-server/v3/adapters/pubmatic"
	"github.com/prebid/prebid-server/v3/adapters/pubnative"
	"github.com/prebid/prebid-server/v3/adapters/pubrise"
	"github.com/prebid/prebid-server/v3/adapters/pulsepoint"
	"github.com/prebid/prebid-server/v3/adapters/pwbid"
	"github.com/prebid/prebid-server/v3/adapters/qt"
	"github.com/prebid/prebid-server/v3/adapters/readpeak"
	"github.com/prebid/prebid-server/v3/adapters/rediads"
	"github.com/prebid/prebid-server/v3/adapters/relevantdigital"
	"github.com/prebid/prebid-server/v3/adapters/resetdigital"
	"github.com/prebid/prebid-server/v3/adapters/revcontent"
	"github.com/prebid/prebid-server/v3/adapters/richaudience"
	"github.com/prebid/prebid-server/v3/adapters/rise"
	"github.com/prebid/prebid-server/v3/adapters/roulax"
	"github.com/prebid/prebid-server/v3/adapters/rtbhouse"
	"github.com/prebid/prebid-server/v3/adapters/rubicon"
	salunamedia "github.com/prebid/prebid-server/v3/adapters/sa_lunamedia"
	"github.com/prebid/prebid-server/v3/adapters/screencore"
	"github.com/prebid/prebid-server/v3/adapters/seedingAlliance"
	"github.com/prebid/prebid-server/v3/adapters/seedtag"
	"github.com/prebid/prebid-server/v3/adapters/sharethrough"
	"github.com/prebid/prebid-server/v3/adapters/silvermob"
	"github.com/prebid/prebid-server/v3/adapters/silverpush"
	"github.com/prebid/prebid-server/v3/adapters/smaato"
	"github.com/prebid/prebid-server/v3/adapters/smartadserver"
	"github.com/prebid/prebid-server/v3/adapters/smarthub"
	"github.com/prebid/prebid-server/v3/adapters/smartrtb"
	"github.com/prebid/prebid-server/v3/adapters/smartx"
	"github.com/prebid/prebid-server/v3/adapters/smartyads"
	"github.com/prebid/prebid-server/v3/adapters/smilewanted"
	"github.com/prebid/prebid-server/v3/adapters/smoot"
	"github.com/prebid/prebid-server/v3/adapters/smrtconnect"
	"github.com/prebid/prebid-server/v3/adapters/sonobi"
	"github.com/prebid/prebid-server/v3/adapters/sovrn"
	"github.com/prebid/prebid-server/v3/adapters/sovrnXsp"
	"github.com/prebid/prebid-server/v3/adapters/sparteo"
	"github.com/prebid/prebid-server/v3/adapters/sspBC"
	"github.com/prebid/prebid-server/v3/adapters/startio"
	"github.com/prebid/prebid-server/v3/adapters/stroeerCore"
	"github.com/prebid/prebid-server/v3/adapters/taboola"
	"github.com/prebid/prebid-server/v3/adapters/tappx"
	"github.com/prebid/prebid-server/v3/adapters/teads"
	"github.com/prebid/prebid-server/v3/adapters/telaria"
	"github.com/prebid/prebid-server/v3/adapters/theadx"
	"github.com/prebid/prebid-server/v3/adapters/thetradedesk"
	"github.com/prebid/prebid-server/v3/adapters/tpmn"
	"github.com/prebid/prebid-server/v3/adapters/tradplus"
	"github.com/prebid/prebid-server/v3/adapters/trafficgate"
	"github.com/prebid/prebid-server/v3/adapters/triplelift"
	"github.com/prebid/prebid-server/v3/adapters/triplelift_native"
	"github.com/prebid/prebid-server/v3/adapters/trustedstack"
	"github.com/prebid/prebid-server/v3/adapters/ucfunnel"
	"github.com/prebid/prebid-server/v3/adapters/undertone"
	"github.com/prebid/prebid-server/v3/adapters/unicorn"
	"github.com/prebid/prebid-server/v3/adapters/unruly"
	"github.com/prebid/prebid-server/v3/adapters/vidazoo"
	"github.com/prebid/prebid-server/v3/adapters/videobyte"
	"github.com/prebid/prebid-server/v3/adapters/videoheroes"
	"github.com/prebid/prebid-server/v3/adapters/vidoomy"
	"github.com/prebid/prebid-server/v3/adapters/visiblemeasures"
	"github.com/prebid/prebid-server/v3/adapters/visx"
	"github.com/prebid/prebid-server/v3/adapters/vox"
	"github.com/prebid/prebid-server/v3/adapters/vrtcal"
	"github.com/prebid/prebid-server/v3/adapters/vungle"
	"github.com/prebid/prebid-server/v3/adapters/xeworks"
	"github.com/prebid/prebid-server/v3/adapters/yahooAds"
	"github.com/prebid/prebid-server/v3/adapters/yandex"
	"github.com/prebid/prebid-server/v3/adapters/yeahmobi"
	"github.com/prebid/prebid-server/v3/adapters/yieldlab"
	"github.com/prebid/prebid-server/v3/adapters/yieldmo"
	"github.com/prebid/prebid-server/v3/adapters/yieldone"
	"github.com/prebid/prebid-server/v3/adapters/zentotem"
	"github.com/prebid/prebid-server/v3/adapters/zeroclickfraud"
	"github.com/prebid/prebid-server/v3/adapters/zeta_global_ssp"
	"github.com/prebid/prebid-server/v3/adapters/zmaticoo"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Adapter registration is kept in this separate file for ease of use and to aid
// in resolving merge conflicts.

func newAdapterBuilders() map[openrtb_ext.BidderName]adapters.Builder {
	return map[openrtb_ext.BidderName]adapters.Builder{
		openrtb_ext.Bidder33Across:          ttx.Builder,
		openrtb_ext.BidderAax:               aax.Builder,
		openrtb_ext.BidderAceex:             aceex.Builder,
		openrtb_ext.BidderAcuityAds:         acuityads.Builder,
		openrtb_ext.BidderAdagio:            adagio.Builder,
		openrtb_ext.BidderAdelement:         adelement.Builder,
		openrtb_ext.BidderAdf:               adf.Builder,
		openrtb_ext.BidderAdgeneration:      adgeneration.Builder,
		openrtb_ext.BidderAdhese:            adhese.Builder,
		openrtb_ext.BidderAdkernel:          adkernel.Builder,
		openrtb_ext.BidderAdkernelAdn:       adkernelAdn.Builder,
		openrtb_ext.BidderAdman:             adman.Builder,
		openrtb_ext.BidderAdmatic:           admatic.Builder,
		openrtb_ext.BidderAdmixer:           admixer.Builder,
		openrtb_ext.BidderAdnuntius:         adnuntius.Builder,
		openrtb_ext.BidderAdOcean:           adocean.Builder,
		openrtb_ext.BidderAdoppler:          adoppler.Builder,
		openrtb_ext.BidderAdot:              adot.Builder,
		openrtb_ext.BidderAdpone:            adpone.Builder,
		openrtb_ext.BidderAdprime:           adprime.Builder,
		openrtb_ext.BidderAdquery:           adquery.Builder,
		openrtb_ext.BidderAdrino:            adrino.Builder,
		openrtb_ext.BidderAdsInteractive:    ads_interactive.Builder,
		openrtb_ext.BidderAdsinteractive:    adsinteractive.Builder,
		openrtb_ext.BidderAdtarget:          adtarget.Builder,
		openrtb_ext.BidderAdtrgtme:          adtrgtme.Builder,
		openrtb_ext.BidderAdtelligent:       adtelligent.Builder,
		openrtb_ext.BidderAdTonos:           adtonos.Builder,
		openrtb_ext.BidderAdUpTech:          aduptech.Builder,
		openrtb_ext.BidderAdvangelists:      advangelists.Builder,
		openrtb_ext.BidderAdverxo:           adverxo.Builder,
		openrtb_ext.BidderAdView:            adview.Builder,
		openrtb_ext.BidderAdxcg:             adxcg.Builder,
		openrtb_ext.BidderAdyoulike:         adyoulike.Builder,
		openrtb_ext.BidderAidem:             aidem.Builder,
		openrtb_ext.BidderAJA:               aja.Builder,
		openrtb_ext.BidderAkcelo:            akcelo.Builder,
		openrtb_ext.BidderAlgorix:           algorix.Builder,
		openrtb_ext.BidderAlkimi:            alkimi.Builder,
		openrtb_ext.BidderAMX:               amx.Builder,
		openrtb_ext.BidderApacdex:           apacdex.Builder,
		openrtb_ext.BidderAppnexus:          appnexus.Builder,
		openrtb_ext.BidderAppush:            appush.Builder,
		openrtb_ext.BidderAso:               aso.Builder,
		openrtb_ext.BidderAudienceNetwork:   audienceNetwork.Builder,
		openrtb_ext.BidderAutomatad:         automatad.Builder,
		openrtb_ext.BidderAvocet:            avocet.Builder,
		openrtb_ext.BidderAxis:              axis.Builder,
		openrtb_ext.BidderAxonix:            axonix.Builder,
		openrtb_ext.BidderBeachfront:        beachfront.Builder,
		openrtb_ext.BidderBeintoo:           beintoo.Builder,
		openrtb_ext.BidderBematterfull:      bematterfull.Builder,
		openrtb_ext.BidderBetween:           between.Builder,
		openrtb_ext.BidderBeyondMedia:       beyondmedia.Builder,
		openrtb_ext.BidderBidmachine:        bidmachine.Builder,
		openrtb_ext.BidderBidmatic:          bidmatic.Builder,
		openrtb_ext.BidderBidmyadz:          bidmyadz.Builder,
		openrtb_ext.BidderBidsCube:          bidscube.Builder,
		openrtb_ext.BidderBidstack:          bidstack.Builder,
		openrtb_ext.BidderBidtheatre:        bidtheatre.Builder,
		openrtb_ext.BidderBigoAd:            bigoad.Builder,
		openrtb_ext.BidderBlasto:            blasto.Builder,
		openrtb_ext.BidderBliink:            bliink.Builder,
		openrtb_ext.BidderBlue:              blue.Builder,
		openrtb_ext.BidderBluesea:           bluesea.Builder,
		openrtb_ext.BidderBmtm:              bmtm.Builder,
		openrtb_ext.BidderBoldwin:           boldwin.Builder,
		openrtb_ext.BidderBrave:             brave.Builder,
		openrtb_ext.BidderBWX:               bwx.Builder,
		openrtb_ext.BidderCadentApertureMX:  cadentaperturemx.Builder,
		openrtb_ext.BidderCcx:               ccx.Builder,
		openrtb_ext.BidderCointraffic:       cointraffic.Builder,
		openrtb_ext.BidderCoinzilla:         coinzilla.Builder,
		openrtb_ext.BidderColossus:          colossus.Builder,
		openrtb_ext.BidderCompass:           compass.Builder,
		openrtb_ext.BidderConcert:           concert.Builder,
		openrtb_ext.BidderConnatix:          connatix.Builder,
		openrtb_ext.BidderConnectAd:         connectad.Builder,
		openrtb_ext.BidderConsumable:        consumable.Builder,
		openrtb_ext.BidderConversant:        conversant.Builder,
		openrtb_ext.BidderCopper6ssp:        copper6ssp.Builder,
		openrtb_ext.BidderCpmstar:           cpmstar.Builder,
		openrtb_ext.BidderCriteo:            criteo.Builder,
		openrtb_ext.BidderCWire:             cwire.Builder,
		openrtb_ext.BidderDatablocks:        datablocks.Builder,
		openrtb_ext.BidderDecenterAds:       decenterads.Builder,
		openrtb_ext.BidderDeepintent:        deepintent.Builder,
		openrtb_ext.BidderDefinemedia:       definemedia.Builder,
		openrtb_ext.BidderDianomi:           dianomi.Builder,
		openrtb_ext.BidderDisplayio:         displayio.Builder,
		openrtb_ext.BidderEdge226:           edge226.Builder,
		openrtb_ext.BidderDmx:               dmx.Builder,
		openrtb_ext.BidderDXKulture:         dxkulture.Builder,
		openrtb_ext.BidderDriftPixel:        driftpixel.Builder,
		openrtb_ext.BidderEmtv:              emtv.Builder,
		openrtb_ext.BidderEmxDigital:        cadentaperturemx.Builder,
		openrtb_ext.BidderEPlanning:         eplanning.Builder,
		openrtb_ext.BidderEpom:              epom.Builder,
		openrtb_ext.BidderEscalax:           escalax.Builder,
		openrtb_ext.BidderExco:              exco.Builder,
		openrtb_ext.BidderEVolution:         evolution.Builder,
		openrtb_ext.BidderFeedAd:            feedad.Builder,
		openrtb_ext.BidderFlatads:           flatads.Builder,
		openrtb_ext.BidderFlipp:             flipp.Builder,
		openrtb_ext.BidderFreewheelSSP:      freewheelssp.Builder,
		openrtb_ext.BidderFWSSP:             fwssp.Builder,
		openrtb_ext.BidderFRVRAdNetwork:     frvradn.Builder,
		openrtb_ext.BidderGamma:             gamma.Builder,
		openrtb_ext.BidderGamoshi:           gamoshi.Builder,
		openrtb_ext.BidderGlobalsun:         globalsun.Builder,
		openrtb_ext.BidderGothamads:         gothamads.Builder,
		openrtb_ext.BidderGrid:              grid.Builder,
		openrtb_ext.BidderGumGum:            gumgum.Builder,
		openrtb_ext.BidderHuaweiAds:         huaweiads.Builder,
		openrtb_ext.BidderImds:              imds.Builder,
		openrtb_ext.BidderImpactify:         impactify.Builder,
		openrtb_ext.BidderImprovedigital:    improvedigital.Builder,
		openrtb_ext.BidderInfyTV:            infytv.Builder,
		openrtb_ext.BidderInMobi:            inmobi.Builder,
		openrtb_ext.BidderInsticator:        insticator.Builder,
		openrtb_ext.BidderInteractiveoffers: interactiveoffers.Builder,
		openrtb_ext.BidderIntertech:         intertech.Builder,
		openrtb_ext.BidderInvibes:           invibes.Builder,
		openrtb_ext.BidderIQX:               iqx.Builder,
		openrtb_ext.BidderIQZone:            iqzone.Builder,
		openrtb_ext.BidderIx:                ix.Builder,
		openrtb_ext.BidderJixie:             jixie.Builder,
		openrtb_ext.BidderKargo:             kargo.Builder,
		openrtb_ext.BidderKayzen:            kayzen.Builder,
		openrtb_ext.BidderKidoz:             kidoz.Builder,
		openrtb_ext.BidderKiviads:           kiviads.Builder,
		openrtb_ext.BidderLmKiviads:         lmkiviads.Builder,
		openrtb_ext.BidderKobler:            kobler.Builder,
		openrtb_ext.BidderKrushmedia:        krushmedia.Builder,
		openrtb_ext.BidderKueezRTB:          kueezrtb.Builder,
		openrtb_ext.BidderLemmadigital:      lemmadigital.Builder,
		openrtb_ext.BidderVungle:            vungle.Builder,
		openrtb_ext.BidderLimelightDigital:  limelightDigital.Builder,
		openrtb_ext.BidderLockerDome:        lockerdome.Builder,
		openrtb_ext.BidderLogan:             logan.Builder,
		openrtb_ext.BidderLogicad:           logicad.Builder,
		openrtb_ext.BidderLoopme:            loopme.Builder,
		openrtb_ext.BidderLoyal:             loyal.Builder,
		openrtb_ext.BidderLunaMedia:         lunamedia.Builder,
		openrtb_ext.BidderMabidder:          mabidder.Builder,
		openrtb_ext.BidderMadSense:          madsense.Builder,
		openrtb_ext.BidderMadvertise:        madvertise.Builder,
		openrtb_ext.BidderMarsmedia:         marsmedia.Builder,
		openrtb_ext.BidderMediafuse:         appnexus.Builder,
		openrtb_ext.BidderMediaGo:           mediago.Builder,
		openrtb_ext.BidderMedianet:          medianet.Builder,
		openrtb_ext.BidderMediasquare:       mediasquare.Builder,
		openrtb_ext.BidderMeloZen:           melozen.Builder,
		openrtb_ext.BidderMetaX:             metax.Builder,
		openrtb_ext.BidderMgid:              mgid.Builder,
		openrtb_ext.BidderMgidX:             mgidX.Builder,
		openrtb_ext.BidderMinuteMedia:       minutemedia.Builder,
		openrtb_ext.BidderMissena:           missena.Builder,
		openrtb_ext.BidderMobfoxpb:          mobfoxpb.Builder,
		openrtb_ext.BidderMobileFuse:        mobilefuse.Builder,
		openrtb_ext.BidderMobkoi:            mobkoi.Builder,
		openrtb_ext.BidderMotorik:           motorik.Builder,
		openrtb_ext.BidderNativo:            nativo.Builder,
		openrtb_ext.BidderNextMillennium:    nextmillennium.Builder,
		openrtb_ext.BidderNexx360:           nexx360.Builder,
		openrtb_ext.BidderNoBid:             nobid.Builder,
		openrtb_ext.BidderOgury:             ogury.Builder,
		openrtb_ext.BidderOms:               oms.Builder,
		openrtb_ext.BidderOneTag:            onetag.Builder,
		openrtb_ext.BidderOpenWeb:           openweb.Builder,
		openrtb_ext.BidderOpenx:             openx.Builder,
		openrtb_ext.BidderOperaads:          operaads.Builder,
		openrtb_ext.BidderOptidigital:       optidigital.Builder,
		openrtb_ext.BidderOraki:             oraki.Builder,
		openrtb_ext.BidderOrbidder:          orbidder.Builder,
		openrtb_ext.BidderOutbrain:          outbrain.Builder,
		openrtb_ext.BidderOwnAdx:            ownadx.Builder,
		openrtb_ext.BidderPangle:            pangle.Builder,
		openrtb_ext.BidderPGAMSsp:           pgamssp.Builder,
		openrtb_ext.BidderPlaydigo:          playdigo.Builder,
		openrtb_ext.BidderPubmatic:          pubmatic.Builder,
		openrtb_ext.BidderPubnative:         pubnative.Builder,
		openrtb_ext.BidderPubrise:           pubrise.Builder,
		openrtb_ext.BidderPulsepoint:        pulsepoint.Builder,
		openrtb_ext.BidderPWBid:             pwbid.Builder,
		openrtb_ext.BidderQT:                qt.Builder,
		openrtb_ext.BidderReadpeak:          readpeak.Builder,
		openrtb_ext.BidderRediads:           rediads.Builder,
		openrtb_ext.BidderRelevantDigital:   relevantdigital.Builder,
		openrtb_ext.BidderResetDigital:      resetdigital.Builder,
		openrtb_ext.BidderRevcontent:        revcontent.Builder,
		openrtb_ext.BidderRichaudience:      richaudience.Builder,
		openrtb_ext.BidderRise:              rise.Builder,
		openrtb_ext.BidderRoulax:            roulax.Builder,
		openrtb_ext.BidderRTBHouse:          rtbhouse.Builder,
		openrtb_ext.BidderRubicon:           rubicon.Builder,
		openrtb_ext.BidderSeedingAlliance:   seedingAlliance.Builder,
		openrtb_ext.BidderSeedtag:           seedtag.Builder,
		openrtb_ext.BidderSaLunaMedia:       salunamedia.Builder,
		openrtb_ext.BidderScreencore:        screencore.Builder,
		openrtb_ext.BidderSharethrough:      sharethrough.Builder,
		openrtb_ext.BidderSilverMob:         silvermob.Builder,
		openrtb_ext.BidderSilverPush:        silverpush.Builder,
		openrtb_ext.BidderSmaato:            smaato.Builder,
		openrtb_ext.BidderSmartAdserver:     smartadserver.Builder,
		openrtb_ext.BidderSmartHub:          smarthub.Builder,
		openrtb_ext.BidderSmartRTB:          smartrtb.Builder,
		openrtb_ext.BidderSmartx:            smartx.Builder,
		openrtb_ext.BidderSmartyAds:         smartyads.Builder,
		openrtb_ext.BidderSmileWanted:       smilewanted.Builder,
		openrtb_ext.BidderSmoot:             smoot.Builder,
		openrtb_ext.BidderSmrtconnect:       smrtconnect.Builder,
		openrtb_ext.BidderSonobi:            sonobi.Builder,
		openrtb_ext.BidderSovrn:             sovrn.Builder,
		openrtb_ext.BidderSovrnXsp:          sovrnXsp.Builder,
		openrtb_ext.BidderSparteo:           sparteo.Builder,
		openrtb_ext.BidderSspBC:             sspBC.Builder,
		openrtb_ext.BidderStartIO:           startio.Builder,
		openrtb_ext.BidderStroeerCore:       stroeerCore.Builder,
		openrtb_ext.BidderTaboola:           taboola.Builder,
		openrtb_ext.BidderTappx:             tappx.Builder,
		openrtb_ext.BidderTeads:             teads.Builder,
		openrtb_ext.BidderTelaria:           telaria.Builder,
		openrtb_ext.BidderTheadx:            theadx.Builder,
		openrtb_ext.BidderTheTradeDesk:      thetradedesk.Builder,
		openrtb_ext.BidderTpmn:              tpmn.Builder,
		openrtb_ext.BidderTradPlus:          tradplus.Builder,
		openrtb_ext.BidderTrafficGate:       trafficgate.Builder,
		openrtb_ext.BidderTriplelift:        triplelift.Builder,
		openrtb_ext.BidderTripleliftNative:  triplelift_native.Builder,
		openrtb_ext.BidderTrustedstack:      trustedstack.Builder,
		openrtb_ext.BidderUcfunnel:          ucfunnel.Builder,
		openrtb_ext.BidderUndertone:         undertone.Builder,
		openrtb_ext.BidderUnicorn:           unicorn.Builder,
		openrtb_ext.BidderUnruly:            unruly.Builder,
		openrtb_ext.BidderVidazoo:           vidazoo.Builder,
		openrtb_ext.BidderVideoByte:         videobyte.Builder,
		openrtb_ext.BidderVideoHeroes:       videoheroes.Builder,
		openrtb_ext.BidderVidoomy:           vidoomy.Builder,
		openrtb_ext.BidderVisibleMeasures:   visiblemeasures.Builder,
		openrtb_ext.BidderVisx:              visx.Builder,
		openrtb_ext.BidderVox:               vox.Builder,
		openrtb_ext.BidderVrtcal:            vrtcal.Builder,
		openrtb_ext.BidderXeworks:           xeworks.Builder,
		openrtb_ext.BidderYahooAds:          yahooAds.Builder,
		openrtb_ext.BidderYandex:            yandex.Builder,
		openrtb_ext.BidderYeahmobi:          yeahmobi.Builder,
		openrtb_ext.BidderYieldlab:          yieldlab.Builder,
		openrtb_ext.BidderYieldmo:           yieldmo.Builder,
		openrtb_ext.BidderYieldone:          yieldone.Builder,
		openrtb_ext.BidderZentotem:          zentotem.Builder,
		openrtb_ext.BidderZeroClickFraud:    zeroclickfraud.Builder,
		openrtb_ext.BidderZetaGlobalSsp:     zeta_global_ssp.Builder,
		openrtb_ext.BidderZmaticoo:          zmaticoo.Builder,
	}
}
