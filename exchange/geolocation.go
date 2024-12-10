package exchange

import (
	"context"
	"errors"

	tcf2 "github.com/prebid/go-gdpr/vendorconsent/tcf2"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/config/countrycode"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/geolocation"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/util/iputil"
)

type GeoLocationResolver interface {
	Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error)
}

type geoLocationResolver struct {
	geoloc geolocation.GeoLocation
	me     metrics.MetricsEngine
}

func (g *geoLocationResolver) Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error) {
	if g.geoloc == nil || ip == "" || country != "" {
		return nil, errors.New("geolocation lookup skipped")
	}
	geoinfo, err := g.geoloc.Lookup(ctx, ip)
	g.me.RecordGeoLocationRequest(err == nil)
	return geoinfo, err
}

func NewGeoLocationResolver(geoloc geolocation.GeoLocation, me metrics.MetricsEngine) *geoLocationResolver {
	return &geoLocationResolver{
		geoloc: geoloc,
		me:     me,
	}
}

func countryFromDevice(device *openrtb2.Device) string {
	if device == nil || device.Geo == nil {
		return ""
	}
	return device.Geo.Country
}

func EnrichGeoLocation(ctx context.Context, req *openrtb_ext.RequestWrapper, account config.Account, geoResolver GeoLocationResolver) (errs []error) {
	if !account.GeoLocation.IsGeoLocationEnabled() {
		return nil
	}

	device := req.BidRequest.Device
	if device == nil {
		return []error{errors.New("device is nil")}
	}

	ip := device.IP
	if ip == "" {
		ip = device.IPv6
	}
	country := countryFromDevice(device)
	geoinfo, err := geoResolver.Lookup(ctx, ip, country)
	if err != nil {
		return []error{err}
	}

	updateDeviceGeo(req.BidRequest, geoinfo)

	return
}

func EnrichGeoLocationWithPrivacy(
	ctx context.Context,
	req *openrtb_ext.RequestWrapper,
	account config.Account,
	geoResolver GeoLocationResolver,
	requestPrivacy *RequestPrivacy,
	tcf2Config gdpr.TCF2ConfigReader,
) (errs []error) {
	if !account.GeoLocation.IsGeoLocationEnabled() {
		return nil
	}

	device := req.BidRequest.Device
	if device == nil {
		return []error{errors.New("device is nil")}
	}

	if requestPrivacy.GDPREnforced {
		return
	}

	country := countryFromDevice(device)
	ip := maybeMaskIP(device, account.Privacy, requestPrivacy, tcf2Config)
	geoinfo, err := geoResolver.Lookup(ctx, ip, country)
	if err != nil {
		return []error{err}
	}

	updateDeviceGeo(req.BidRequest, geoinfo)

	return
}

func maybeMaskIP(device *openrtb2.Device, accountPrivacy config.AccountPrivacy, requestPrivacy *RequestPrivacy, tcf2Config gdpr.TCF2ConfigReader) string {
	if device == nil {
		return ""
	}

	shouldBeMasked := shouldMaskIP(requestPrivacy, tcf2Config)
	if device.IP != "" {
		if shouldBeMasked {
			return privacy.ScrubIP(device.IP, accountPrivacy.IPv4Config.AnonKeepBits, iputil.IPv4BitSize)
		}
		return device.IP
	} else if device.IPv6 != "" {
		if shouldBeMasked {
			return privacy.ScrubIP(device.IPv6, accountPrivacy.IPv6Config.AnonKeepBits, iputil.IPv6BitSize)
		}
		return device.IPv6
	}
	return ""
}

func shouldMaskIP(requestPrivacy *RequestPrivacy, tcf2Config gdpr.TCF2ConfigReader) bool {
	if requestPrivacy.COPPAEnforced || requestPrivacy.LMTEnforced {
		return true
	}
	if requestPrivacy.ParsedConsent != nil {
		cm, ok := requestPrivacy.ParsedConsent.(tcf2.ConsentMetadata)
		return ok && !tcf2Config.FeatureOneEnforced() && !cm.SpecialFeatureOptIn(1)
	}
	return false
}

func updateDeviceGeo(req *openrtb2.BidRequest, geoinfo *geolocation.GeoInfo) {
	if req.Device == nil || geoinfo == nil {
		return
	}

	device := *req.Device
	if device.Geo == nil {
		device.Geo = &openrtb2.Geo{}
	}

	geo := device.Geo
	if alpha3 := countrycode.ToAlpha3(geoinfo.Country); alpha3 != "" {
		geo.Country = alpha3
	}
	if geoinfo.Region != "" {
		geo.Region = geoinfo.Region
	}
	if offset, err := geolocation.TimezoneToUTCOffset(geoinfo.TimeZone); err == nil {
		geo.UTCOffset = int64(offset)
	}

	req.Device = &device
}

type NilGeoLocationResolver struct{}

func (g *NilGeoLocationResolver) Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error) {
	return &geolocation.GeoInfo{}, nil
}
