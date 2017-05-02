package cache

import "fmt"

// DummyCache dummy config that will echo back results
type DummyCache struct {
}

// NewDummyCache create new config
func NewDummyCache() *DummyCache {

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

	d := &Domain{
		Domain: key,
	}
	return d, nil
}

// GetAccount echos back the account
func (c *DummyCache) GetAccount(key string) (*Account, error) {

	d := &Account{
		ID: key,
	}
	return d, nil
}
