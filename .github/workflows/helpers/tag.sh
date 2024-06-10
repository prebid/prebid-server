currentTag=$(git describe --abbrev=0 --tags)
echo "Current release tag ${currentTag}"

echo ${currentTag} | grep -q "^v\?[0-9]\+\.[0-9]\+\.[0-9]\+$"
if [ $? -ne 0 ]; then
echo "Current tag format won't let us compute the new tag name. Required format v[0-9]\+\.[0-9]\+\.[0-9]\+"
exit 1
fi

if [[ "${currentTag:0:1}" != "v" ]]; then
currentTag="v${currentTag}"
fi

nextTag=''
releaseType=${{ inputs.releaseType }}
if [ $releaseType == "major" ]; then
    # PBS-GO skipped the v1.0.0 major release - https://github.com/prebid/prebid-server/issues/3068
    # If the current tag is v0.x.x, the script sets the next release tag to v2.0.0
    # Otherwise, the script increments the major version by 1 and sets the minor and patch versions to zero
    # For example, v2.x.x will be incremented to v3.0.0
    major=$(echo "${currentTag}" | awk -F. '{gsub(/^v/, "", $1); if($1 == 0) $1=2; else $1+=1; print $1}')
    nextTag="v${major}.0.0"
elif [ $releaseType == "minor" ]; then
    # Increment minor version and reset patch version
    nextTag=$(echo "${currentTag}" | awk -F. '{OFS="."; $2+=1; $3=0; print $0}')
else
# Increment patch version
nextTag=$(echo "${currentTag}" | awk -F. '{OFS="."; $3+=1; print $0}')
fi

if [ ${{ inputs.debug }} == 'true' ]; then
echo "running workflow in debug mode, next ${releaseType} tag: ${nextTag}"
else
git tag $nextTag
git push origin $nextTag
echo "tag=${nextTag}" >> $GITHUB_OUTPUT
fi

echo "::set-output name=releaseTag::${nextTag}"