package exchange

import (
	"github.com/prebid/prebid-server/v2/adapters"
	ttx "github.com/prebid/prebid-server/v2/adapters/33across"
	"github.com/prebid/prebid-server/v2/adapters/aax"
	"github.com/prebid/prebid-server/v2/adapters/aceex"
	"github.com/prebid/prebid-server/v2/adapters/acuityads"
	"github.com/prebid/prebid-server/v2/adapters/adelement"
	"github.com/prebid/prebid-server/v2/adapters/adf"
	"github.com/prebid/prebid-server/v2/adapters/adgeneration"
	"github.com/prebid/prebid-server/v2/adapters/adhese"
	"github.com/prebid/prebid-server/v2/adapters/adkernel"
	"github.com/prebid/prebid-server/v2/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/v2/adapters/adman"
	"github.com/prebid/prebid-server/v2/adapters/admatic"
	"github.com/prebid/prebid-server/v2/adapters/admixer"
	"github.com/prebid/prebid-server/v2/adapters/adnuntius"
	"github.com/prebid/prebid-server/v2/adapters/adocean"
	"github.com/prebid/prebid-server/v2/adapters/adoppler"
	"github.com/prebid/prebid-server/v2/adapters/adot"
	"github.com/prebid/prebid-server/v2/adapters/adpone"
	"github.com/prebid/prebid-server/v2/adapters/adprime"
	"github.com/prebid/prebid-server/v2/adapters/adquery"
	"github.com/prebid/prebid-server/v2/adapters/adrino"
	"github.com/prebid/prebid-server/v2/adapters/adsinteractive"
	"github.com/prebid/prebid-server/v2/adapters/adtarget"
	"github.com/prebid/prebid-server/v2/adapters/adtelligent"
	"github.com/prebid/prebid-server/v2/adapters/adtrgtme"
	"github.com/prebid/prebid-server/v2/adapters/advangelists"
	"github.com/prebid/prebid-server/v2/adapters/adview"
	"github.com/prebid/prebid-server/v2/adapters/adxcg"
	"github.com/prebid/prebid-server/v2/adapters/adyoulike"
	"github.com/prebid/prebid-server/v2/adapters/aidem"
	"github.com/prebid/prebid-server/v2/adapters/aja"
	"github.com/prebid/prebid-server/v2/adapters/algorix"
	"github.com/prebid/prebid-server/v2/adapters/alkimi"
	"github.com/prebid/prebid-server/v2/adapters/amx"
	"github.com/prebid/prebid-server/v2/adapters/apacdex"
	"github.com/prebid/prebid-server/v2/adapters/appnexus"
	"github.com/prebid/prebid-server/v2/adapters/appush"
	"github.com/prebid/prebid-server/v2/adapters/aso"
	"github.com/prebid/prebid-server/v2/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/v2/adapters/automatad"
	"github.com/prebid/prebid-server/v2/adapters/avocet"
	"github.com/prebid/prebid-server/v2/adapters/axis"
	"github.com/prebid/prebid-server/v2/adapters/axonix"
	"github.com/prebid/prebid-server/v2/adapters/beachfront"
	"github.com/prebid/prebid-server/v2/adapters/beintoo"
	"github.com/prebid/prebid-server/v2/adapters/bematterfull"
	"github.com/prebid/prebid-server/v2/adapters/between"
	"github.com/prebid/prebid-server/v2/adapters/beyondmedia"
	"github.com/prebid/prebid-server/v2/adapters/bidmachine"
	"github.com/prebid/prebid-server/v2/adapters/bidmyadz"
	"github.com/prebid/prebid-server/v2/adapters/bidscube"
	"github.com/prebid/prebid-server/v2/adapters/bidstack"
	"github.com/prebid/prebid-server/v2/adapters/blasto"
	"github.com/prebid/prebid-server/v2/adapters/bliink"
	"github.com/prebid/prebid-server/v2/adapters/blue"
	"github.com/prebid/prebid-server/v2/adapters/bluesea"
	"github.com/prebid/prebid-server/v2/adapters/bmtm"
	"github.com/prebid/prebid-server/v2/adapters/boldwin"
	"github.com/prebid/prebid-server/v2/adapters/brave"
	"github.com/prebid/prebid-server/v2/adapters/bwx"
	cadentaperturemx "github.com/prebid/prebid-server/v2/adapters/cadent_aperture_mx"
	"github.com/prebid/prebid-server/v2/adapters/ccx"
	"github.com/prebid/prebid-server/v2/adapters/cointraffic"
	"github.com/prebid/prebid-server/v2/adapters/coinzilla"
	"github.com/prebid/prebid-server/v2/adapters/colossus"
	"github.com/prebid/prebid-server/v2/adapters/compass"
	"github.com/prebid/prebid-server/v2/adapters/concert"
	"github.com/prebid/prebid-server/v2/adapters/connectad"
	"github.com/prebid/prebid-server/v2/adapters/consumable"
	"github.com/prebid/prebid-server/v2/adapters/conversant"
	"github.com/prebid/prebid-server/v2/adapters/cpmstar"
	"github.com/prebid/prebid-server/v2/adapters/criteo"
	"github.com/prebid/prebid-server/v2/adapters/cwire"
	"github.com/prebid/prebid-server/v2/adapters/datablocks"
	"github.com/prebid/prebid-server/v2/adapters/decenterads"
	"github.com/prebid/prebid-server/v2/adapters/deepintent"
	"github.com/prebid/prebid-server/v2/adapters/definemedia"
	"github.com/prebid/prebid-server/v2/adapters/dianomi"
	"github.com/prebid/prebid-server/v2/adapters/dmx"
	"github.com/prebid/prebid-server/v2/adapters/driftpixel"
	"github.com/prebid/prebid-server/v2/adapters/dxkulture"
	evolution "github.com/prebid/prebid-server/v2/adapters/e_volution"
	"github.com/prebid/prebid-server/v2/adapters/edge226"
	"github.com/prebid/prebid-server/v2/adapters/emtv"
	"github.com/prebid/prebid-server/v2/adapters/eplanning"
	"github.com/prebid/prebid-server/v2/adapters/epom"
	"github.com/prebid/prebid-server/v2/adapters/flipp"
	"github.com/prebid/prebid-server/v2/adapters/freewheelssp"
	"github.com/prebid/prebid-server/v2/adapters/frvradn"
	"github.com/prebid/prebid-server/v2/adapters/gamma"
	"github.com/prebid/prebid-server/v2/adapters/gamoshi"
	"github.com/prebid/prebid-server/v2/adapters/globalsun"
	"github.com/prebid/prebid-server/v2/adapters/gothamads"
	"github.com/prebid/prebid-server/v2/adapters/grid"
	"github.com/prebid/prebid-server/v2/adapters/gumgum"
	"github.com/prebid/prebid-server/v2/adapters/huaweiads"
	"github.com/prebid/prebid-server/v2/adapters/imds"
	"github.com/prebid/prebid-server/v2/adapters/impactify"
	"github.com/prebid/prebid-server/v2/adapters/improvedigital"
	"github.com/prebid/prebid-server/v2/adapters/infytv"
	"github.com/prebid/prebid-server/v2/adapters/inmobi"
	"github.com/prebid/prebid-server/v2/adapters/interactiveoffers"
	"github.com/prebid/prebid-server/v2/adapters/invibes"
	"github.com/prebid/prebid-server/v2/adapters/iqx"
	"github.com/prebid/prebid-server/v2/adapters/iqzone"
	"github.com/prebid/prebid-server/v2/adapters/ix"
	"github.com/prebid/prebid-server/v2/adapters/jixie"
	"github.com/prebid/prebid-server/v2/adapters/kargo"
	"github.com/prebid/prebid-server/v2/adapters/kayzen"
	"github.com/prebid/prebid-server/v2/adapters/kidoz"
	"github.com/prebid/prebid-server/v2/adapters/kiviads"
	"github.com/prebid/prebid-server/v2/adapters/krushmedia"
	"github.com/prebid/prebid-server/v2/adapters/lemmadigital"
	"github.com/prebid/prebid-server/v2/adapters/limelightDigital"
	lmkiviads "github.com/prebid/prebid-server/v2/adapters/lm_kiviads"
	"github.com/prebid/prebid-server/v2/adapters/lockerdome"
	"github.com/prebid/prebid-server/v2/adapters/logan"
	"github.com/prebid/prebid-server/v2/adapters/logicad"
	"github.com/prebid/prebid-server/v2/adapters/loyal"
	"github.com/prebid/prebid-server/v2/adapters/lunamedia"
	"github.com/prebid/prebid-server/v2/adapters/mabidder"
	"github.com/prebid/prebid-server/v2/adapters/madvertise"
	"github.com/prebid/prebid-server/v2/adapters/marsmedia"
	"github.com/prebid/prebid-server/v2/adapters/mediago"
	"github.com/prebid/prebid-server/v2/adapters/medianet"
	"github.com/prebid/prebid-server/v2/adapters/mgid"
	"github.com/prebid/prebid-server/v2/adapters/mgidX"
	"github.com/prebid/prebid-server/v2/adapters/minutemedia"
	"github.com/prebid/prebid-server/v2/adapters/mobfoxpb"
	"github.com/prebid/prebid-server/v2/adapters/mobilefuse"
	"github.com/prebid/prebid-server/v2/adapters/motorik"
	"github.com/prebid/prebid-server/v2/adapters/nextmillennium"
	"github.com/prebid/prebid-server/v2/adapters/nobid"
	"github.com/prebid/prebid-server/v2/adapters/oms"
	"github.com/prebid/prebid-server/v2/adapters/onetag"
	"github.com/prebid/prebid-server/v2/adapters/openweb"
	"github.com/prebid/prebid-server/v2/adapters/openx"
	"github.com/prebid/prebid-server/v2/adapters/operaads"
	"github.com/prebid/prebid-server/v2/adapters/orbidder"
	"github.com/prebid/prebid-server/v2/adapters/outbrain"
	"github.com/prebid/prebid-server/v2/adapters/ownadx"
	"github.com/prebid/prebid-server/v2/adapters/pangle"
	"github.com/prebid/prebid-server/v2/adapters/pgamssp"
	"github.com/prebid/prebid-server/v2/adapters/playdigo"
	"github.com/prebid/prebid-server/v2/adapters/pubmatic"
	"github.com/prebid/prebid-server/v2/adapters/pubnative"
	"github.com/prebid/prebid-server/v2/adapters/pulsepoint"
	"github.com/prebid/prebid-server/v2/adapters/pwbid"
	"github.com/prebid/prebid-server/v2/adapters/readpeak"
	"github.com/prebid/prebid-server/v2/adapters/relevantdigital"
	"github.com/prebid/prebid-server/v2/adapters/revcontent"
	"github.com/prebid/prebid-server/v2/adapters/richaudience"
	"github.com/prebid/prebid-server/v2/adapters/rise"
	"github.com/prebid/prebid-server/v2/adapters/roulax"
	"github.com/prebid/prebid-server/v2/adapters/rtbhouse"
	"github.com/prebid/prebid-server/v2/adapters/rubicon"
	salunamedia "github.com/prebid/prebid-server/v2/adapters/sa_lunamedia"
	"github.com/prebid/prebid-server/v2/adapters/screencore"
	"github.com/prebid/prebid-server/v2/adapters/seedingAlliance"
	"github.com/prebid/prebid-server/v2/adapters/sharethrough"
	"github.com/prebid/prebid-server/v2/adapters/silvermob"
	"github.com/prebid/prebid-server/v2/adapters/silverpush"
	"github.com/prebid/prebid-server/v2/adapters/smaato"
	"github.com/prebid/prebid-server/v2/adapters/smartadserver"
	"github.com/prebid/prebid-server/v2/adapters/smarthub"
	"github.com/prebid/prebid-server/v2/adapters/smartrtb"
	"github.com/prebid/prebid-server/v2/adapters/smartx"
	"github.com/prebid/prebid-server/v2/adapters/smartyads"
	"github.com/prebid/prebid-server/v2/adapters/smilewanted"
	"github.com/prebid/prebid-server/v2/adapters/smrtconnect"
	"github.com/prebid/prebid-server/v2/adapters/sonobi"
	"github.com/prebid/prebid-server/v2/adapters/sovrn"
	"github.com/prebid/prebid-server/v2/adapters/sovrnXsp"
	"github.com/prebid/prebid-server/v2/adapters/sspBC"
	"github.com/prebid/prebid-server/v2/adapters/stroeerCore"
	"github.com/prebid/prebid-server/v2/adapters/taboola"
	"github.com/prebid/prebid-server/v2/adapters/tappx"
	"github.com/prebid/prebid-server/v2/adapters/teads"
	"github.com/prebid/prebid-server/v2/adapters/telaria"
	"github.com/prebid/prebid-server/v2/adapters/theadx"
	"github.com/prebid/prebid-server/v2/adapters/tpmn"
	"github.com/prebid/prebid-server/v2/adapters/trafficgate"
	"github.com/prebid/prebid-server/v2/adapters/triplelift"
	"github.com/prebid/prebid-server/v2/adapters/triplelift_native"
	"github.com/prebid/prebid-server/v2/adapters/trustedstack"
	"github.com/prebid/prebid-server/v2/adapters/ucfunnel"
	"github.com/prebid/prebid-server/v2/adapters/undertone"
	"github.com/prebid/prebid-server/v2/adapters/unicorn"
	"github.com/prebid/prebid-server/v2/adapters/unruly"
	"github.com/prebid/prebid-server/v2/adapters/vidazoo"
	"github.com/prebid/prebid-server/v2/adapters/videobyte"
	"github.com/prebid/prebid-server/v2/adapters/videoheroes"
	"github.com/prebid/prebid-server/v2/adapters/vidoomy"
	"github.com/prebid/prebid-server/v2/adapters/visiblemeasures"
	"github.com/prebid/prebid-server/v2/adapters/visx"
	"github.com/prebid/prebid-server/v2/adapters/vox"
	"github.com/prebid/prebid-server/v2/adapters/vrtcal"
	"github.com/prebid/prebid-server/v2/adapters/vungle"
	"github.com/prebid/prebid-server/v2/adapters/xeworks"
	"github.com/prebid/prebid-server/v2/adapters/yahooAds"
	"github.com/prebid/prebid-server/v2/adapters/yandex"
	"github.com/prebid/prebid-server/v2/adapters/yeahmobi"
	"github.com/prebid/prebid-server/v2/adapters/yieldlab"
	"github.com/prebid/prebid-server/v2/adapters/yieldmo"
	"github.com/prebid/prebid-server/v2/adapters/yieldone"
	"github.com/prebid/prebid-server/v2/adapters/zeroclickfraud"
	"github.com/prebid/prebid-server/v2/adapters/zeta_global_ssp"
	"github.com/prebid/prebid-server/v2/adapters/zmaticoo"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// Adapter registration is kept in this separate file for ease of use and to aid
// in resolving merge conflicts.

func newAdapterBuilders() map[openrtb_ext.BidderName]adapters.Builder {
	return map[openrtb_ext.BidderName]adapters.Builder{
		openrtb_ext.Bidder33Across:          ttx.Builder,
		openrtb_ext.BidderAax:               aax.Builder,
		openrtb_ext.BidderAceex:             aceex.Builder,
		openrtb_ext.BidderAcuityAds:         acuityads.Builder,
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
		openrtb_ext.BidderAdsinteractive:    adsinteractive.Builder,
		openrtb_ext.BidderAdtarget:          adtarget.Builder,
		openrtb_ext.BidderAdtrgtme:          adtrgtme.Builder,
		openrtb_ext.BidderAdtelligent:       adtelligent.Builder,
		openrtb_ext.BidderAdvangelists:      advangelists.Builder,
		openrtb_ext.BidderAdView:            adview.Builder,
		openrtb_ext.BidderAdxcg:             adxcg.Builder,
		openrtb_ext.BidderAdyoulike:         adyoulike.Builder,
		openrtb_ext.BidderAidem:             aidem.Builder,
		openrtb_ext.BidderAJA:               aja.Builder,
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
		openrtb_ext.BidderBidmyadz:          bidmyadz.Builder,
		openrtb_ext.BidderBidsCube:          bidscube.Builder,
		openrtb_ext.BidderBidstack:          bidstack.Builder,
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
		openrtb_ext.BidderConnectAd:         connectad.Builder,
		openrtb_ext.BidderConsumable:        consumable.Builder,
		openrtb_ext.BidderConversant:        conversant.Builder,
		openrtb_ext.BidderCpmstar:           cpmstar.Builder,
		openrtb_ext.BidderCriteo:            criteo.Builder,
		openrtb_ext.BidderCWire:             cwire.Builder,
		openrtb_ext.BidderDatablocks:        datablocks.Builder,
		openrtb_ext.BidderDecenterAds:       decenterads.Builder,
		openrtb_ext.BidderDeepintent:        deepintent.Builder,
		openrtb_ext.BidderDefinemedia:       definemedia.Builder,
		openrtb_ext.BidderDianomi:           dianomi.Builder,
		openrtb_ext.BidderEdge226:           edge226.Builder,
		openrtb_ext.BidderDmx:               dmx.Builder,
		openrtb_ext.BidderDXKulture:         dxkulture.Builder,
		openrtb_ext.BidderDriftPixel:        driftpixel.Builder,
		openrtb_ext.BidderEmtv:              emtv.Builder,
		openrtb_ext.BidderEmxDigital:        cadentaperturemx.Builder,
		openrtb_ext.BidderEPlanning:         eplanning.Builder,
		openrtb_ext.BidderEpom:              epom.Builder,
		openrtb_ext.BidderEVolution:         evolution.Builder,
		openrtb_ext.BidderFlipp:             flipp.Builder,
		openrtb_ext.BidderFreewheelSSP:      freewheelssp.Builder,
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
		openrtb_ext.BidderInteractiveoffers: interactiveoffers.Builder,
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
		openrtb_ext.BidderKrushmedia:        krushmedia.Builder,
		openrtb_ext.BidderLemmadigital:      lemmadigital.Builder,
		openrtb_ext.BidderVungle:            vungle.Builder,
		openrtb_ext.BidderLimelightDigital:  limelightDigital.Builder,
		openrtb_ext.BidderLockerDome:        lockerdome.Builder,
		openrtb_ext.BidderLogan:             logan.Builder,
		openrtb_ext.BidderLogicad:           logicad.Builder,
		openrtb_ext.BidderLoyal:             loyal.Builder,
		openrtb_ext.BidderLunaMedia:         lunamedia.Builder,
		openrtb_ext.BidderMabidder:          mabidder.Builder,
		openrtb_ext.BidderMadvertise:        madvertise.Builder,
		openrtb_ext.BidderMarsmedia:         marsmedia.Builder,
		openrtb_ext.BidderMediafuse:         appnexus.Builder,
		openrtb_ext.BidderMediaGo:           mediago.Builder,
		openrtb_ext.BidderMedianet:          medianet.Builder,
		openrtb_ext.BidderMgid:              mgid.Builder,
		openrtb_ext.BidderMgidX:             mgidX.Builder,
		openrtb_ext.BidderMinuteMedia:       minutemedia.Builder,
		openrtb_ext.BidderMobfoxpb:          mobfoxpb.Builder,
		openrtb_ext.BidderMobileFuse:        mobilefuse.Builder,
		openrtb_ext.BidderMotorik:           motorik.Builder,
		openrtb_ext.BidderNextMillennium:    nextmillennium.Builder,
		openrtb_ext.BidderNoBid:             nobid.Builder,
		openrtb_ext.BidderOms:               oms.Builder,
		openrtb_ext.BidderOneTag:            onetag.Builder,
		openrtb_ext.BidderOpenWeb:           openweb.Builder,
		openrtb_ext.BidderOpenx:             openx.Builder,
		openrtb_ext.BidderOperaads:          operaads.Builder,
		openrtb_ext.BidderOrbidder:          orbidder.Builder,
		openrtb_ext.BidderOutbrain:          outbrain.Builder,
		openrtb_ext.BidderOwnAdx:            ownadx.Builder,
		openrtb_ext.BidderPangle:            pangle.Builder,
		openrtb_ext.BidderPGAMSsp:           pgamssp.Builder,
		openrtb_ext.BidderPlaydigo:          playdigo.Builder,
		openrtb_ext.BidderPubmatic:          pubmatic.Builder,
		openrtb_ext.BidderPubnative:         pubnative.Builder,
		openrtb_ext.BidderPulsepoint:        pulsepoint.Builder,
		openrtb_ext.BidderPWBid:             pwbid.Builder,
		openrtb_ext.BidderReadpeak:          readpeak.Builder,
		openrtb_ext.BidderRelevantDigital:   relevantdigital.Builder,
		openrtb_ext.BidderRevcontent:        revcontent.Builder,
		openrtb_ext.BidderRichaudience:      richaudience.Builder,
		openrtb_ext.BidderRise:              rise.Builder,
		openrtb_ext.BidderRoulax:            roulax.Builder,
		openrtb_ext.BidderRTBHouse:          rtbhouse.Builder,
		openrtb_ext.BidderRubicon:           rubicon.Builder,
		openrtb_ext.BidderSeedingAlliance:   seedingAlliance.Builder,
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
		openrtb_ext.BidderSmrtconnect:       smrtconnect.Builder,
		openrtb_ext.BidderSonobi:            sonobi.Builder,
		openrtb_ext.BidderSovrn:             sovrn.Builder,
		openrtb_ext.BidderSovrnXsp:          sovrnXsp.Builder,
		openrtb_ext.BidderSspBC:             sspBC.Builder,
		openrtb_ext.BidderStroeerCore:       stroeerCore.Builder,
		openrtb_ext.BidderTaboola:           taboola.Builder,
		openrtb_ext.BidderTappx:             tappx.Builder,
		openrtb_ext.BidderTeads:             teads.Builder,
		openrtb_ext.BidderTelaria:           telaria.Builder,
		openrtb_ext.BidderTheadx:            theadx.Builder,
		openrtb_ext.BidderTpmn:              tpmn.Builder,
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
		openrtb_ext.BidderZeroClickFraud:    zeroclickfraud.Builder,
		openrtb_ext.BidderZetaGlobalSsp:     zeta_global_ssp.Builder,
		openrtb_ext.BidderZmaticoo:          zmaticoo.Builder,
	}
}
