package cache

type DummyCache struct {
}

func NewDummyCache() *DummyCache {

	return &DummyCache{}
}

func (c *DummyCache) Close() {
}

func (c *DummyCache) GetConfig(key string) (string, error) {
	return "", nil
}

func (c *DummyCache) GetDomain(key string) (*Domain, error) {

	d := &Domain{
		Domain: key,
	}
	return d, nil
}

func (c *DummyCache) GetAccount(key string) (*Account, error) {

	d := &Account{
		ID: key,
	}
	return d, nil
}
