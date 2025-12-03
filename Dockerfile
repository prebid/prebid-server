FROM ubuntu:22.04 AS build
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends wget ca-certificates
WORKDIR /tmp
RUN wget https://dl.google.com/go/go1.24.0.linux-amd64.tar.gz && \
    tar -xf go1.24.0.linux-amd64.tar.gz && \
    mv go /usr/local
RUN mkdir -p /app/prebid-server/
WORKDIR /app/prebid-server/
ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
ENV GOPROXY="https://proxy.golang.org"

# Installing gcc as cgo uses it to build native code of some modules
RUN apt-get update && \
    apt-get install -y --no-install-recommends git gcc build-essential curl tar gzip && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# CGO must be enabled because some modules depend on native C code
ENV CGO_ENABLED 1

# MaxMind database download (required for mile.floors module)
ARG MAXMIND_ACCOUNT_ID
ARG MAXMIND_LICENSE_KEY
COPY ./ ./

# Download MaxMind database (required for mile.floors module)
RUN set -e && \
    echo "=== MaxMind Download Step ===" && \
    echo "MAXMIND_LICENSE_KEY length: ${#MAXMIND_LICENSE_KEY}" && \
    echo "MAXMIND_ACCOUNT_ID length: ${#MAXMIND_ACCOUNT_ID}" && \
    if [ -z "$MAXMIND_LICENSE_KEY" ]; then \
        echo "ERROR: MAXMIND_LICENSE_KEY is required but not provided"; \
        exit 1; \
    fi && \
    echo "Downloading MaxMind GeoLite2-Country database..." && \
    chmod +x scripts/download-maxmind.sh && \
    MAXMIND_ACCOUNT_ID="$MAXMIND_ACCOUNT_ID" MAXMIND_LICENSE_KEY="$MAXMIND_LICENSE_KEY" ./scripts/download-maxmind.sh /app/prebid-server/GeoLite2-Country.mmdb && \
    echo "=== Verifying download ===" && \
    ls -la /app/prebid-server/GeoLite2-Country.mmdb && \
    echo "MaxMind database downloaded successfully"
RUN go mod tidy
RUN go mod vendor
ARG TEST="true"
# RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
RUN go build -mod=vendor .

FROM ubuntu:22.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-server .
RUN chmod a+xr prebid-server
COPY static static/
COPY stored_requests/data stored_requests/data
RUN chmod -R a+r static/ stored_requests/data

# Installing libatomic1 as it is a runtime dependency for some modules
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates mtr libatomic1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy MaxMind database from build stage
# The database is required for mile.floors module, so fail if it's missing
RUN mkdir -p /opt/maxmind && \
    if [ -f /usr/local/bin/GeoLite2-Country.mmdb ]; then \
        cp /usr/local/bin/GeoLite2-Country.mmdb /opt/maxmind/GeoLite2-Country.mmdb && \
        chmod 644 /opt/maxmind/GeoLite2-Country.mmdb && \
        echo "MaxMind database copied to /opt/maxmind/GeoLite2-Country.mmdb" && \
        ls -lh /opt/maxmind/GeoLite2-Country.mmdb; \
    else \
        echo "ERROR: MaxMind database not found at /usr/local/bin/GeoLite2-Country.mmdb" && \
        echo "The mile.floors module requires this database. Build will fail." && \
        exit 1; \
    fi

RUN addgroup --system --gid 2001 prebidgroup && adduser --system --uid 1001 --ingroup prebidgroup prebid
USER prebid
EXPOSE 8000
EXPOSE 6060
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
