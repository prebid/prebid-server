#!/bin/bash

set -ex

BRANCHES=( "master" "development" )

cp /secrets/reposync/github_id_rsa ~/.ssh/github_id_rsa
# Configure SSH key
mkdir -p ~/.ssh
cat << EOF > ~/.ssh/config
Host github.com
 IdentityFile ~/.ssh/github_id_rsa
EOF

cat ~/.ssh/config
mkdir -p /code
cd /code

# Clone from google code
gcloud auth activate-service-account --key-file=/secrets/reposync/google.json

if [[ ! -d /code/prebid-cloudops ]]; then
    gcloud source repos clone prebid-cloudops
fi

cd prebid-cloudops
git fetch origin

# Add github remote & fetch latest
git remote add github git@github.com:newscorp-ghfb/prebid-server.git || true
chmod 0400 ~/.ssh/github_id_rsa
ls -ltr ~/.ssh/github_id_rsa

git fetch github

# Mirror branches
for b in ${BRANCHES[*]}; do
    echo "mirroring $b"
    git checkout github/${b}
    git push --force origin HEAD:refs/heads/${b}
done
