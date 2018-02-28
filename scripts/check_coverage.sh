#!/bin/bash

#cleanup
finish() {
  if [ -d ".cover" ]; then
    rm -rf .cover
  fi
}

trap finish EXIT ERR INT TERM

# If the coverage script runs without errors, then make sure that it meets the min code coverage
OUTPUT=`./scripts/coverage.sh`
if [ "$?" = "0" ]; then
  while read -r LINE; do
    echo -e "$LINE"
    if [[ $LINE =~ "%" ]]; then
      PERCENT=$(echo "$LINE"|cut -d: -f2-|cut -d% -f1|cut -d. -f1|tr -d ' ')
      if [[ $PERCENT -lt 30 ]]; then
        echo "Package has less than 30% code coverage. Run ./scripts/coverage.sh --html to see a detailed coverage report, and add tests to improve your coverage"
        exit 1
      fi
    fi
  done <<< "$OUTPUT"

# Fixes #315. If it has errors, print those and exit with an error code
else
  while read -r LINE; do
    echo -e "$LINE"
  done <<< "$OUTPUT"
  exit 1
fi
