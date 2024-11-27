package maxmind

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync/atomic"

	"github.com/prebid/prebid-server/v3/geolocation"

	geoip2 "github.com/oschwald/geoip2-golang"
)

const Vendor = "maxmind"

const DatabaseFileName = "GeoLite2-City.mmdb"

// GeoLocation implementations geolocation.GeoLocation interface.
type GeoLocation struct {
	reader atomic.Pointer[geoip2.Reader]
}

func (g *GeoLocation) Lookup(_ context.Context, ipAddress string) (*geolocation.GeoInfo, error) {
	ip := net.ParseIP(ipAddress)
	if len(ip) == 0 {
		return nil, geolocation.ErrLookupIPInvalid
	}

	reader := g.reader.Load()
	if reader == nil {
		return nil, geolocation.ErrDatabaseUnavailable
	}

	record, err := reader.City(ip)
	if err != nil {
		return nil, err
	}

	info := &geolocation.GeoInfo{
		Vendor:    Vendor,
		Continent: record.Continent.Code,
		Country:   record.Country.IsoCode,
		Zip:       record.Postal.Code,
		Lat:       record.Location.Latitude,
		Lon:       record.Location.Longitude,
		TimeZone:  record.Location.TimeZone,
	}
	if len(record.Subdivisions) > 0 {
		info.Region = record.Subdivisions[0].IsoCode
	}
	if len(record.City.Names) > 0 {
		info.City = record.City.Names["en"]
	}
	return info, nil
}

// SetDataPath loads data and updates the reader.
func (g *GeoLocation) SetDataPath(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		// io.EOF and other errors
		if err != nil {
			return errors.New("failed to read tar file: " + err.Error())
		}

		if header.Name == DatabaseFileName {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, tarReader); err != nil {
				return err
			}
			reader, err := geoip2.FromBytes(buf.Bytes())
			if err != nil {
				return err
			}
			g.reader.Store(reader)
			return nil
		}
	}
}
