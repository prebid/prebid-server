#!/bin/bash -e

prefix="v"
to_major=0
to_minor=208
to_patch=0
upgrade_version="$prefix$to_major.$to_minor.$to_patch"

attempt=4

usage="
Script starts or continues prebid upgrade to version set in 'to_minor' variable. Workspace is at /tmp/prebid-server and /tmp/pbs-patch

    ./upgrade-pbs.sh [--restart]

    --restart   Restart the upgrade (deletes /tmp/prebid-server and /tmp/pbs-patch)
    -h          Help

TODO:
    - paramertrize the script
    - create ci branch PR
    - create header-bidding PR"

RESTART=0
for i in "$@"; do
  case $i in
    --restart)
      RESTART=1
      shift
      ;;
    -h)
      echo "$usage"
      exit 0
      ;;
  esac
done

# --- start ---
CHECKLOG=/tmp/pbs-patch/checkpoints.log

trap 'clear_log' EXIT

log () {
  printf "\n$(date): $1\n"
}

clear_log() {
    major=0
    minor=0
    patch=0
    get_current_tag_version major minor patch
    current_fork_at_version="$major.$minor.$patch"

    if [ "$current_fork_at_version" == "$upgrade_version" ] ; then
        log "Upgraded to $current_fork_at_version"
        rm -f "$CHECKLOG"
    
        log "Last validation before creating PR"
        go_mod
        checkpoint_run "./validate.sh --race 5"
        go_discard

        set +e
        log "Commit final go.mod and go.sum"
        git commit go.mod go.sum --amend --no-edit
        set -e
    else
        log "Exiting with failure!!!"
        exit 1
    fi
}

get_current_tag_version() {
    log "get_current_tag_version $*"

    local -n _major=$1
    local -n _minor=$2
    local -n _patch=$3

    # script will always start from start if origin/master is used.
    # common_commit=$(git merge-base prebid-upstream/master origin/master)
    # log "Common commit b/w prebid-upstream/master origin/master: $common_commit"

    # remove origin for master to continue from last fixed tag's rebase.
    common_commit=$(git merge-base prebid-upstream/master master)
    log "Common commit b/w prebid-upstream/master master: $common_commit"

    current_version=$(git tag --points-at $common_commit)
    if [[ $current_version == v* ]] ; then
        log "Current Version: $current_version"
    else
        log "Failed to detected current version. Abort."
        exit 1
        # abort
        # cd prebid-server; git rebase --abort;cd -
    fi

    IFS='.' read -r -a _current_version <<< "$current_version"
    _major=${_current_version[0]}
    _minor=${_current_version[1]}
    _patch=${_current_version[2]}
}

clone_repo() {
    if [ -d "/tmp/prebid-server" ]; then
        log "Code already cloned. Attempting to continue the upgrade!!!"
    else
        log "Cloning repo at /tmp"
        cd /tmp
        git clone https://github.com/PubMatic-OpenWrap/prebid-server.git
        cd prebid-server

        git remote add prebid-upstream https://github.com/prebid/prebid-server.git
        git remote -v
        git fetch --all --tags --prune
    fi
}

checkout_branch() {
    set +e
    git checkout tags/$_upgrade_version -b $tag_base_branch_name
    # git push origin $tag_base_branch_name

    git checkout -b $upgrade_branch_name
    # git push origin $upgrade_branch_name

    set -e
#    if [ "$?" -ne 0 ]
#    then
#        log "Failed to create branch $upgrade_branch_name. Already working on it???"
#        exit 1
#    fi
}

cmd_exe() {
    cmd=$*
    if ! $cmd; then
        log "Failure!!! creating checkpoint $cmd"
        echo "$cmd" > $CHECKLOG
        exit 1
    fi
}

checkpoint_run() {
    cmd=$*
    if [ -f $CHECKLOG ] ; then
        if grep -q "$cmd" "$CHECKLOG"; then
            log "Retry this checkpoint: $cmd"
            rm "$CHECKLOG"
        elif grep -q "./validate.sh --race 5" "$CHECKLOG"; then
            log "Special checkpoint. ./validate.sh --race 5 failed for last tag update. Hence, only fixes are expected in successfully upgraded branch. (change in func() def, wrong conflict resolve, etc)"
            cmd_exe $cmd
            rm "$CHECKLOG"
        else
            log "Skip this checkpoint: $cmd"
            return
        fi
    fi
    cmd_exe $cmd
}

go_mod() {
    go mod download all
    go mod tidy
    go mod tidy
    go mod download all
}

go_discard() {
    # discard local changes if any. manual validate, compile, etc
    # git checkout master go.mod
    # git checkout master go.sum
    git checkout go.mod go.sum
}

# --- main ---

if [ "$RESTART" -eq "1" ]; then
    log "Restarting the upgrade: rm -rf /tmp/prebid-server /tmp/pbs-patch/"
    rm -rf /tmp/prebid-server /tmp/pbs-patch/ 
    mkdir -p /tmp/pbs-patch/
fi

log "Final Upgrade Version: $upgrade_version"
log "Attempt: $attempt"

checkpoint_run clone_repo
cd /tmp/prebid-server
log "At $(pwd)"

# code merged in master
# if [ "$RESTART" -eq "1" ]; then
#     # TODO: commit this in origin/master,ci and remove it from here.
#     git merge --squash origin/UOE-7610-1-upgrade.sh
#     git commit --no-edit
# fi

major=0
minor=0
patch=0

get_current_tag_version major minor patch
current_fork_at_version="$major.$minor.$patch"
git diff tags/$current_fork_at_version..origin/master > /tmp/pbs-patch/current_ow_patch-$current_fork_at_version-origin_master-$attempt.diff

((minor++))
log "Starting with version split major:$major, minor:$minor, patch:$patch"

# how to validate with this code
# if [ "$RESTART" -eq "1" ]; then
#     # Solving go.mod and go.sum conflicts would be easy at last as we would need to only pick the OW-patch entries rather than resolving conflict for every version
#     log "Using latest go.mod and go.sum. Patch OW changes at last"
#     git checkout tags/$current_fork_at_version go.mod
#     git checkout tags/$current_fork_at_version go.sum
#     git commit go.mod go.sum -m "[upgrade-start-checkpoint] tags/$current_fork_at_version go.mod go.sum"
# fi

log "Checking if last failure was for test case. Need this to pick correct"
go_mod
checkpoint_run "./validate.sh --race 5"
go_discard

log "Starting upgrade loop..."
while [ "$minor" -le "$to_minor" ]; do
    # _upgrade_version="$prefix$major.$minor.$patch"
    _upgrade_version="$major.$minor.$patch"
    ((minor++))

    log "Starting upgrade to version $_upgrade_version"

    tag_base_branch_name=prebid_$_upgrade_version-$attempt-tag
    upgrade_branch_name=prebid_$_upgrade_version-$attempt
    log "Reference tag branch: $tag_base_branch_name"
    log "Upgrade branch: $upgrade_branch_name"

    checkpoint_run checkout_branch

    checkpoint_run git merge master --no-edit
    # Use `git commit --amend --no-edit` if you had to fix test cases, etc for wrong merge conflict resolve, etc.
    log "Validating the master merge into current tag. Fix and commit changes if required. Use 'git commit --amend --no-edit' for consistency"
    go_mod
    checkpoint_run "./validate.sh --race 5"
    go_discard

    checkpoint_run git checkout master
    checkpoint_run git merge $upgrade_branch_name --no-edit

    log "Generating patch file at /tmp/pbs-patch/ for $_upgrade_version"
    git diff tags/$_upgrade_version..master > /tmp/pbs-patch/new_ow_patch_$upgrade_version-master-1.diff
done

# TODO:
# diff tags/v0.192.0..origin/master
# diff tags/v0.207.0..prebid_v0.207.0

# TODO: UPDATE HEADER-BIDDING GO-MOD


# TODO: automate go.mod conflicts
# go mod edit -replace github.com/prebid/prebid-server=./
# go mod edit -replace github.com/mxmCherry/openrtb/v15=github.com/PubMatic-OpenWrap/openrtb/v15@v15.0.0
# go mod edit -replace github.com/beevik/etree=github.com/PubMatic-OpenWrap/etree@latest
