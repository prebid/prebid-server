# Automated Tests

This project uses GitHub Actions to make sure that every PR passes automated tests.
To reproduce these tests locally, use:

```
./validate --nofmt --cov
```

## Writing Tests

Tests for `some-file.go` should be placed in the file `some-file_test.go` in the same package.
For more info on how to write tests in Go, see [the Go docs](https://golang.org/pkg/testing/).

## Adapter Tests

If your adapter makes HTTP calls using standard JSON, you should use the
[RunJSONBidderTest](https://github.com/prebid/prebid-server/blob/master/adapters/adapterstest/test_json.go#L50) function.

This will be much more thorough, convenient, maintainable, and reusable than writing standard Go tests
for your adapter.

## Concurrency Tests

Code which creates new goroutines should include tests which thoroughly exercise its concurrent behavior.
The names of functions which test concurrency should start with `TestRace`. For example `TestRaceAuction` or `TestRaceCurrency`.

The `./validate.sh` script will run these using the [Race Detector](https://golang.org/doc/articles/race_detector.html).
