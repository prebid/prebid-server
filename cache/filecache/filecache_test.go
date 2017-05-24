package filecache

import (
	"io/ioutil"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestFileCache(t *testing.T) {
	fcf := fileCacheFile{
		Domains:  []string{"one.com", "two.com", "three.com"},
		Apps:     []string{"com.app.one", "com.app.two", "com.app.three"},
		Accounts: []string{"account1", "account2", "account3"},
		Configs: []fileConfig{
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

	dataCache, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	d, err := dataCache.Domains().Get("one.com")
	if err != nil {
		t.Fatal(err)
	}

	if d.Domain != "one.com" {
		t.Error("fetched invalid domain")
	}

	d, err = dataCache.Domains().Get("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}

	app, err := dataCache.Apps().Get("com.app.one")
	if err != nil {
		t.Fatal(err)
	}

	if app.Bundle != "com.app.one" {
		t.Error("fetched invalid app")
	}

	app, err = dataCache.Apps().Get("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}

	a, err := dataCache.Accounts().Get("account1")
	if err != nil {
		t.Fatal(err)
	}

	if a.ID != "account1" {
		t.Error("fetched invalid domain")
	}

	a, err = dataCache.Accounts().Get("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}

	c, err := dataCache.Config().Get("one")
	if err != nil {
		t.Fatal(err)
	}

	if c != "config1" {
		t.Error("fetched invalid domain")
	}

	c, err = dataCache.Config().Get("abc123")
	if err == nil {
		t.Error("domain should not exist in cache")
	}
}
