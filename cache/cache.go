package cache

type Domain struct {
	Domain string `json:"domain"`
}

type App struct {
	Bundle string `json:"bundle"`
}

type Account struct {
	ID string `json:"id"`
}

type Configuration struct {
	Type string `json:"type"` // required
}

type Cache interface {
	Configure(*Configuration) error
	Close() error

	Accounts() interface {
		Get(string) (*Account, error)
		Set(*Account) error
	}
	Apps() interface {
		Get(string) (*App, error)
		Set(*App) error
	}
	Config() interface {
		Get(string) (string, error)
		Set(string) error
	}
	Domains() interface {
		Get(string) (*Domain, error)
		Set(*Domain) error
	}
}
