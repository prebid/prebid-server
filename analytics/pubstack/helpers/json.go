package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/analytics"
)

// JsonifyAuctionObject helpers to serialize auction into json line
func JsonifyAuctionObject(ao *analytics.AuctionObject, scope string) ([]byte, error) {
	type alias analytics.AuctionObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(ao),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("auction object badly formed %v", err)
}

func JsonifyVideoObject(vo *analytics.VideoObject, scope string) ([]byte, error) {
	type alias analytics.VideoObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(vo),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("video object badly formed %v", err)
}

func JsonifyCookieSync(cso *analytics.CookieSyncObject, scope string) ([]byte, error) {
	type alias analytics.CookieSyncObject

	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(cso),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("cookie sync object badly formed %v", err)
}

func JsonifySetUIDObject(so *analytics.SetUIDObject, scope string) ([]byte, error) {
	type alias analytics.SetUIDObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(so),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("set UID object badly formed %v", err)
}

func JsonifyAmpObject(ao *analytics.AmpObject, scope string) ([]byte, error) {
	type alias analytics.AmpObject
	b, err := json.Marshal(&struct {
		Scope string `json:"scope"`
		*alias
	}{
		Scope: scope,
		alias: (*alias)(ao),
	})

	if err == nil {
		b = append(b, byte('\n'))
		return b, nil
	}
	return nil, fmt.Errorf("amp object badly formed %v", err)
}
