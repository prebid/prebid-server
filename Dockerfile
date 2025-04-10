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
    apt-get install -y --no-install-recommends git gcc build-essential && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# CGO must be enabled because some modules depend on native C code
ENV CGO_ENABLED 1
COPY ./ ./

# Installing WURFL compile-time dependencies if libwurfl package is present
RUN if ls modules/scientiamobile/wurfl_devicedetection/libwurfl/libwurfl*.deb 1> /dev/null 2>&1; then \
      dpkg -i modules/scientiamobile/wurfl_devicedetection/libwurfl/libwurfl*.deb; \
    fi

RUN go mod tidy
RUN go mod vendor
# Accept Go build tags as arguments (default: none)
ARG GO_BUILD_TAGS=""
ARG TEST="true"
RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
RUN go build $GO_BUILD_TAGS -mod=vendor -ldflags "-X github.com/prebid/prebid-server/v3/version.Ver=`git describe --tags | sed 's/^v//'` -X github.com/prebid/prebid-server/v3/version.Rev=`git rev-parse HEAD`" .

FROM ubuntu:22.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-server .
RUN chmod a+xr prebid-server
COPY static static/
COPY stored_requests/data stored_requests/data
RUN chmod -R a+r static/ stored_requests/data

# Installing WURFL runtime dependencies if libwurfl package is present
COPY modules/scientiamobile/wurfl_devicedetection/libwurfl/ /tmp/wurfl
RUN if ls /tmp/wurfl/libwurfl*.deb 1> /dev/null 2>&1; then \
      dpkg -i /tmp/wurfl/libwurfl*.deb; \
      apt-get update && \
      apt-get install -y curl && \
      apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
      rm -rf /tmp/wurfl; \
    fi

# Installing libatomic1 as it is a runtime dependency for some modules
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates mtr libatomic1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN addgroup --system --gid 2001 prebidgroup && adduser --system --uid 1001 --ingroup prebidgroup prebid
USER prebid
EXPOSE 8000
EXPOSE 6060
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
