FROM debian:stretch-slim

RUN set -ex; \
    apt-get update \
    && apt-get install --yes gnupg git lsof

ADD files/apt/google-cloud-sdk.list /etc/apt/sources.list.d/
ADD files/apt/apt-key.gpg /tmp
RUN apt-key add /tmp/apt-key.gpg && apt-get update && apt-get install --yes google-cloud-sdk

ADD files/ssh/known_hosts /root/.ssh/known_hosts

ADD files/sync.sh files/sync-forever.sh /
CMD /sync.sh
