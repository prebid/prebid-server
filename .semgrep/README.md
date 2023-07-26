# Semgrep Test

Running semgrep unit tests:
```bash
semgrep --test
```


Running single semgrep rules against adapter code:
```bash
semgrep --config=./adapter/{rule}.yml ../adapters/
```

Running all semgrep rules simultaneously:
```bash
semgrep --config=./adapter ../adapters/
```
