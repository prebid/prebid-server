package exchange

import (
	"context"
	"errors"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/geolocation"
	"github.com/prebid/prebid-server/v3/geolocation/countrycodemapper"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type GeoLocationResolver interface {
	Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error)
}

type geoLocationResolver struct {
	geoloc geolocation.GeoLocation
}

func (g *geoLocationResolver) Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error) {
	if g.geoloc == nil || ip == "" || country != "" {
		return nil, errors.New("geolocation lookup skipped")
	}
	geoinfo, err := g.geoloc.Lookup(ctx, ip)
	return geoinfo, err
}

func NewGeoLocationResolver(geoloc geolocation.GeoLocation) *geoLocationResolver {
	return &geoLocationResolver{
		geoloc: geoloc,
	}
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

func countryFromDevice(device *openrtb2.Device) string {
	if device == nil || device.Geo == nil {
		return ""
	}
	return device.Geo.Country
}

func updateDeviceGeo(req *openrtb2.BidRequest, geoinfo *geolocation.GeoInfo) {
	if req.Device == nil || geoinfo == nil {
		return
	}

	device := *req.Device
	if device.Geo == nil {
		device.Geo = &openrtb2.Geo{}
	}

	if alpha3 := countrycodemapper.MapToAlpha3(geoinfo.Country); alpha3 != "" {
		device.Geo.Country = alpha3
	}
	if geoinfo.Region != "" {
		if geoinfo.Country == "US" {
			device.Geo.Region = strings.TrimPrefix(geoinfo.Region, "US-")
		} else {
			device.Geo.Region = geoinfo.Region
		}
	}
	if offset, err := geolocation.TimezoneToUTCOffset(geoinfo.TimeZone); err == nil {
		device.Geo.UTCOffset = int64(offset)
	}

	req.Device = &device
}

type NilGeoLocationResolver struct{}

func (g *NilGeoLocationResolver) Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error) {
	return &geolocation.GeoInfo{}, nil
}
