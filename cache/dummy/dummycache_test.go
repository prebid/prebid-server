package dummy

import "testing"

func TestDummyCache(t *testing.T) {

	c := New()

	domain, err := c.GetDomain("one.com")
	if err != nil {
		t.Fatal(err)
	}

	if domain.Domain != "one.com" {
		t.Error("Wrong domain returned")
	}

	app, err := c.GetApp("com.app.one")
	if err != nil {
		t.Fatal(err)
	}

	if app.Bundle != "com.app.one" {
		t.Error("Wrong app returned")
	}

	account, err := c.GetAccount("account1")
	if err != nil {
		t.Fatal(err)
	}

	if account.ID != "account1" {
		t.Error("Wrong account returned")
	}

	cfg, err := c.GetConfig("config")
	if err == nil {
		t.Error("Dummy configs are not supported")
	}

	if cfg != "" {
		t.Error("Dummy config should return empty string")
	}

}
