package cache

type Domain struct {
	Domain string `json:"domain"`
}

type App struct {
	Bundle string `json:"bundle"`
}

type Account struct {
	ID               string `json:"id"`
	PriceGranularity string `json:"price_granularity"`
}

type Configuration struct {
	Type string `json:"type"` // required
}

type Cache interface {
	Close() error
	Accounts() AccountsService
	Apps() AppsService
	Config() ConfigService
	Domains() DomainsService
}

type AccountsService interface {
	Get(string) (*Account, error)
	Set(*Account) error
}

type AppsService interface {
	Get(string) (*App, error)
	Set(*App) error
}

type ConfigService interface {
	Get(string) (string, error)
	Set(string, string) error
}

type DomainsService interface {
	Get(string) (*Domain, error)
	Set(*Domain) error
}
