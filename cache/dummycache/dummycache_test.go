package dummycache

import "testing"

func TestDummyCache(t *testing.T) {

	c, _ := New()

	domain, err := c.Domains().Get("one.com")
	if err != nil {
		t.Fatal(err)
	}

	if domain.Domain != "one.com" {
		t.Error("Wrong domain returned")
	}

	app, err := c.Apps().Get("com.app.one")
	if err != nil {
		t.Fatal(err)
	}

	if app.Bundle != "com.app.one" {
		t.Error("Wrong app returned")
	}

	account, err := c.Accounts().Get("account1")
	if err != nil {
		t.Fatal(err)
	}

	if account.ID != "account1" {
		t.Error("Wrong account returned")
	}

	if err := c.Config().Set("config", "abc123"); err != nil {
		t.Errorf("Dummy config should return nil")
	}

	cfg, err := c.Config().Get("config")
	if err != nil {
		t.Error("Dummy configs should be supported")
	}

	if cfg != "abc123" {
		t.Error("Dummy config did not return back expected string")
	}

}
