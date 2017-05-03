package cache

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
)

// FileCache is a file backed cache
type FileCache struct {
	Configs  map[string]string
	Domains  map[string]bool
	Accounts map[string]bool
}

type fileCacheConfig struct {
	ID     string `yaml:"id"`
	Config string `yaml:"config"`
}

type fileCacheFile struct {
	Configs  []fileCacheConfig `yaml:"configs"`
	Domains  []string          `yaml:"domains"`
	Accounts []string          `yaml:"accounts"`
}

// NewFileCache will load the file into memory
func NewFileCache(filename string) (*FileCache, error) {

	if glog.V(2) {
		glog.Infof("Reading inventory urls from %s", filename)
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Parsing filecache YAML")
	}

	var u fileCacheFile
	err = yaml.Unmarshal(b, &u)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Building URL map")
	}

	fc := &FileCache{}

	fc.Configs = make(map[string]string, len(u.Configs))
	for _, config := range u.Configs {
		fc.Configs[config.ID] = config.Config
	}
	glog.Infof("Loaded %d configs", len(u.Configs))

	fc.Domains = make(map[string]bool, len(u.Domains))
	for _, domain := range u.Domains {
		fc.Domains[domain] = true
	}
	glog.Infof("Loaded %d domains", len(u.Domains))

	fc.Accounts = make(map[string]bool, len(u.Accounts))
	for _, Account := range u.Accounts {
		fc.Accounts[Account] = true
	}
	glog.Infof("Loaded %d accounts", len(u.Accounts))

	return fc, nil
}

// Close does nothing
func (c *FileCache) Close() {
}

// GetConfig will return config from memory if it exists
func (c *FileCache) GetConfig(key string) (string, error) {
	cfg, ok := c.Configs[key]
	if !ok {
		return "", fmt.Errorf("Not found")
	}

	return cfg, nil
}

// GetDomain will return Domain from memory if it exists
func (c *FileCache) GetDomain(key string) (*Domain, error) {

	d := &Domain{
		Domain: key,
	}

	_, ok := c.Domains[key]
	if !ok {
		return nil, fmt.Errorf("Not found")
	}

	return d, nil
}

// GetAccount will return Account from memory if it exists
func (c *FileCache) GetAccount(key string) (*Account, error) {

	d := &Account{
		ID: key,
	}

	_, ok := c.Accounts[key]
	if !ok {
		return nil, fmt.Errorf("Not found")
	}

	return d, nil
}
