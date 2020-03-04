#!/bin/bash

set -ex

BRANCHES=( "master" )

# Configure SSH key
mkdir -p ~/.ssh
cat << EOF > ~/.ssh/config
Host github.com
 IdentityFile /secrets/reposync/github_id_rsa
EOF

mkdir -p /code
cd /code

# Clone from google code
gcloud auth activate-service-account --key-file=/secrets/reposync/google.json

if [[ ! -d /code/prebid ]]; then
    gcloud source repos clone prebid-cloudops
fi

cd prebid-cloudops
git fetch origin

# Add github remote & fetch latest
git remote add github git@github.com:newscorp-ghfb/prebid-server.git || true
git fetch github

# Mirror branches
for b in ${BRANCHES[*]}; do
    echo "mirroring $b"
    git checkout github/${b}
    git push --force origin HEAD:refs/heads/${b}
done
