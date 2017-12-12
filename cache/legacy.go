package cache

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Domain struct {
	Domain string `json:"domain"`
}

type App struct {
	Bundle string `json:"bundle"`
}

type Account struct {
	ID               string                       `json:"id"`
	PriceGranularity openrtb_ext.PriceGranularity `json:"price_granularity"`
}

type Configuration struct {
	Type string `json:"type"` // required
}

type Cache interface {
	Close() error
	Accounts() AccountsService
	Config() ConfigService
}

type AccountsService interface {
	Get(string) (*Account, error)
	Set(*Account) error
}

type ConfigService interface {
	Get(string) (string, error)
	Set(string, string) error
}
