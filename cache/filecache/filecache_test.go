package filecache

import (
	"io/ioutil"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestFileCache(t *testing.T) {
	fcf := fileCacheFile{
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

	a, err := dataCache.Accounts().Get("account1")
	if err != nil {
		t.Fatal(err)
	}

	if a.ID != "account1" {
		t.Error("fetched invalid account")
	}

	a, err = dataCache.Accounts().Get("abc123")
	if err == nil {
		t.Error("account should not exist in cache")
	}

	c, err := dataCache.Config().Get("one")
	if err != nil {
		t.Fatal(err)
	}

	if c != "config1" {
		t.Error("fetched invalid config")
	}

	c, err = dataCache.Config().Get("abc123")
	if err == nil {
		t.Error("config should not exist in cache")
	}
}
