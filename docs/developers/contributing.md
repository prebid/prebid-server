# Contributing

## Development

During development, run the unit tests with:

```bash
./validate.sh
```
New submissions *must* include unit tests. Bugfixes should include a test which prevents that bug from being re-introduced in the future.

## Pull Requests

When your changes are complete, run the tests with code coverage and strict format checking:

```bash
./validate.sh --nofmt --cov
```

All pull requests must have 90% coverage. View your coverage report with:

```bash
./scripts/coverage.sh --html
```

When you're ready, [submit a Pull Request](https://help.github.com/articles/creating-a-pull-request/) against
[our GitHub repository](https://github.com/prebid/prebid-server/compare).
These same tests will be run with [Travis CI](https://travis-ci.com/).

If the tests pass locally, but fail on your PR, make sure to `git pull` the latest code from `master`.

**Note**: We also have some [known intermittent failures](https://github.com/prebid/prebid-server/issues/103).
          If the tests still fail after pulling `master`, don't worry about it. We'll re-run them when we review your PR.
