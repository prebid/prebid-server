#!/bin/bash

set -e

CHECKCOV=false

#cleanup
finish() {

  if [ -d ".cover" ]; then
    rm -rf .cover
  fi
}

trap finish EXIT ERR INT TERM

#start script logic
OUTPUT=`./scripts/coverage.sh`

while read -r LINE; do
  if [[ $LINE =~ "%" ]]; then
    PERCENT=$(echo "$LINE"|cut -d: -f2-|cut -d% -f1|cut -d. -f1|tr -d ' ')
    if [[ $PERCENT -lt 20 ]]; then
      CHECKCOV=true
    fi
  fi
done < "$OUTPUT"

if $CHECKCOV; then
  echo "Detected at least one package had less than 20% code coverage.  Please review results below or from your terminal run ./scripts/coverage.sh --html for more detailed results"
  exit 1
fi
