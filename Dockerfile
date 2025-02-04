FROM ubuntu:20.04 AS build
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y wget
WORKDIR /tmp
RUN wget https://dl.google.com/go/go1.23.4.linux-amd64.tar.gz && \
    tar -xf go1.23.4.linux-amd64.tar.gz && \
    mv go /usr/local
RUN mkdir -p /app/prebid-server/
WORKDIR /app/prebid-server/
ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
ENV GOPROXY="https://proxy.golang.org"

# Installing gcc as cgo uses it to build native code of some modules
RUN apt-get update && \
    apt-get install -y git gcc && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# CGO must be enabled because some modules depend on native C code
ENV CGO_ENABLED 1
COPY ./ ./
RUN go mod tidy
RUN go mod vendor
ARG TEST="true"
#RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
#RUN go build -mod=vendor -ldflags "-X github.com/prebid/prebid-server/v3/version.Ver=`git describe --tags | sed 's/^v//'` -X github.com/prebid/prebid-server/v3/version.Rev=`git rev-parse HEAD`" .
RUN go build -mod=vendor

FROM ubuntu:20.04 AS release
LABEL maintainer="hans.hjort@xandr.com"
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-server .
RUN chmod a+xr prebid-server
COPY static static/
COPY stored_requests/data stored_requests/data
RUN chmod -R a+r static/ stored_requests/data

# Installing libatomic1 as it is a runtime dependency for some modules
RUN apt-get update && \
    apt-get install -y ca-certificates mtr libatomic1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN adduser prebid_user
USER prebid_user
EXPOSE 8000
EXPOSE 6060
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
