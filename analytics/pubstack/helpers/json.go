package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/analytics"
)

func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*analytics.AuctionObject
	}{
		Scope:         scope,
		AuctionObject: ao,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("auction object badly formed %v", err)
}

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) ([]byte, error) {
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*analytics.VideoObject
	}{
		Scope:       scope,
		VideoObject: vo,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("video object badly formed %v", err)
}

func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) ([]byte, error) {
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*analytics.CookieSyncObject
	}{
		Scope:            scope,
		CookieSyncObject: cso,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("cookie sync object badly formed %v", err)
}

func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) ([]byte, error) {
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*analytics.SetUIDObject
	}{
		Scope:        scope,
		SetUIDObject: so,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("set UID object badly formed %v", err)
}

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) ([]byte, error) {
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*analytics.AmpObject
	}{
		Scope:     scope,
		AmpObject: ao,
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("amp object badly formed %v", err)
}
