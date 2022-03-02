ARG ROOT_IMAGE
###################
### Base Image  ###
###################
FROM ${ROOT_IMAGE} as baseimage

# Install OS-level language locales
ENV DEBIAN_FRONTEND=noninteractive LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8 \
    APP_ROOT=/go/src/github.com/tapjoy/tpe_prebid_service APP_USER=webuser APP_USER_UID=1001

RUN apt-get update -q &&\
    apt-get install -yq --no-install-recommends locales locales-all &&\
    locale-gen $LANG && update-locale LANG=$LANG &&\
    rm -rf /var/lib/apt/lists/* /tmp/*

# We install OS-level dependencies we need to work with the project
RUN apt-get update -q &&\
    apt-get install -y --no-install-recommends  ca-certificates curl dnsutils iftop git gnupg2 htop iotop iproute2 jq less lsof rng-tools sysstat vim &&\
    rm -rf /var/lib/apt/lists/* /tmp/*

ENV CHAMBER_VERSION="v2.10.8"
RUN curl -o /usr/local/bin/chamber "https://tj-ops.s3.amazonaws.com/k8s-production/chamber-${CHAMBER_VERSION}-linux-$(uname -m)" &&\
    chmod +x /usr/local/bin/chamber &&\
    chamber version

RUN mkdir -p ${APP_ROOT} &&\
    useradd -m -u ${APP_USER_UID} ${APP_USER} &&\
    usermod -G staff -a ${APP_USER} &&\
    chown -R ${APP_USER}:${APP_USER} ${APP_ROOT}

WORKDIR ${APP_ROOT}

###################
# Build-time prep #
###################

FROM baseimage as artifact-prep

# Copy local-to-builder files and folders into current directory (WORKDIR) of the container
ADD . .

# Remove untracked files and folders
# Run artifact preparation steps (e.g. geoip, bundle install, etc)
# Clean up
RUN git clean -fxd &&\
    make artifact-prep &&\
    chown -R ${APP_USER}:${APP_USER} ${APP_ROOT} &&\
    rm -rf .git /tmp/*

###################
# Artifact target #
###################

FROM baseimage as artifact

USER ${APP_USER}

COPY --from=artifact-prep ${APP_ROOT} /project
WORKDIR /project

# Gut check
RUN ./tpe_prebid_service --help

# @see https://github.com/Tapjoy/tpe_prebid_service/blob/9d0e0c46bb90a4fb818305b06d55725817882697/config/config.go#L570-L571
## Viper port
EXPOSE 8000
## Admin port
EXPOSE 6060
