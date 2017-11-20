package empty_fetcher

import "testing"

func TestErrorLength(t *testing.T) {
	fetcher := EmptyFetcher()

	cfgs, errs := fetcher.GetConfigs([]string{"a", "b"})
	if len(cfgs) != 0 {
		t.Errorf("The empty fetcher should never return configs. Got %d", len(cfgs))
	}
	if len(errs) != 2 {
		t.Errorf("The empty fetcher should return 2 errors. Got %d", len(errs))
	}
}
