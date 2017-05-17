package dummy

import "fmt"

// DummyCache dummy config that will echo back results
type DummyCache struct {
}

// New creates new DummyCache
func New() *DummyCache {

	return &DummyCache{}
}

// Close nop
func (c *DummyCache) Close() {
}

// GetConfig not supported, always returns and error
func (c *DummyCache) GetConfig(key string) (string, error) {
	return "", fmt.Errorf("Not supported")
}

// GetDomain echos back the domain
func (c *DummyCache) GetDomain(key string) (*Domain, error) {
	return &Domain{
		Domain: key,
	}, nil
}

// GetAccount echos back the account
func (c *DummyCache) GetAccount(key string) (*Account, error) {
	return &Account{
		ID: key,
	}, nil
}

// GetApp echos back the app
func (c *DummyCache) GetApp(bundle string) (*App, error) {
	return &App{
		Bundle: bundle,
	}, nil
}
