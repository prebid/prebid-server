package cache

type Domain struct {
	Domain string
}

type App struct {
	Bundle string
}

type Account struct {
	ID string
}

type Cache interface {
	GetDomain(domain string) (*Domain, error)
	GetApp(bundle string) (*App, error)
	GetAccount(id string) (*Account, error)
	GetConfig(id string) (string, error)
	Close()
}
