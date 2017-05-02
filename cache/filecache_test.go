package cache

import (
	"io/ioutil"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestFileCache(t *testing.T) {
	fcf := fileCacheFile{
		Domains:  []string{"one.com", "two.com", "three.com"},
		Accounts: []string{"account1", "account2", "account3"},
		Configs: []fileCacheConfig{
			{
				ID:     "one",
				Config: "config1",
			}, {
				ID:     "two",
				Config: "config2",
			}, {
				ID:     "three",
				Config: "config3",
			},
		},
	}

	bytes, err := yaml.Marshal(&fcf)
	if err != nil {
		t.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile("", "filecache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(bytes); err != nil {
		t.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	dataCache, err := NewFileCache(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	d, err := dataCache.GetDomain("one.com")
	if err != nil {
		t.Fatal(err)
	}

	if d.Domain != "one.com" {
		t.Error("fetched invalid domain")
	}

	d, err = dataCache.GetDomain("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}

	a, err := dataCache.GetAccount("account1")
	if err != nil {
		t.Fatal(err)
	}

	if a.ID != "account1" {
		t.Error("fetched invalid domain")
	}

	a, err = dataCache.GetAccount("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}

	c, err := dataCache.GetConfig("one")
	if err != nil {
		t.Fatal(err)
	}

	if c != "config1" {
		t.Error("fetched invalid domain")
	}

	c, err = dataCache.GetConfig("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}
}
