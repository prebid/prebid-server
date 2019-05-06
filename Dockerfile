FROM ubuntu:18.04 AS build
WORKDIR /go/src/github.com/prebid/prebid-server/
RUN \
     sed -i 's/# deb/deb/g' /etc/apt/sources.list && \
     apt-get update && \
     apt-get -y upgrade && \
     apt-get install -y ca-certificates mtr && \
     apt-get install -y git golang go-dep && \
     apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV GOPATH /go
ENV CGO_ENABLED 0
COPY ./ ./
RUN dep ensure && \
    go build .

FROM ubuntu:18.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /go/src/github.com/prebid/prebid-server/prebid-server .
COPY static static/
COPY stored_requests/data stored_requests/data
EXPOSE 8000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
