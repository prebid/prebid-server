#!/bin/bash


COV_MIN=30

#cleanup
finish() {

  if [ -d ".cover" ]; then
    rm -rf .cover
  fi
}

trap finish EXIT ERR INT TERM

#start script logic
OUTPUT=`./scripts/coverage.sh`
if [[ "$?" -ne "0" ]]; then
  echo -e "$OUTPUT"
  exit 1
fi

while IFS= read -r LINE; do
  echo -e "$LINE"
  if [[ $LINE =~ "%" ]]; then
    PERCENT=$(echo "$LINE"|cut -d: -f2-|cut -d% -f1|cut -d. -f1|tr -d ' ')
    if [[ $PERCENT -lt $COV_MIN ]]; then
      echo "Package has less than ${COV_MIN}% code coverage. Run ./scripts/coverage.sh --html to see a detailed coverage report, and add tests to improve your coverage"
      exit 1
    fi
  fi
done <<< "$OUTPUT"
