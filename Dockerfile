FROM ubuntu:14.04 AS build
WORKDIR /go/src/github.com/prebid/prebid-server/
RUN \
     sed -i 's/# \(.*multiverse$\)/\1/g' /etc/apt/sources.list && \
     apt-get update && \
     apt-get -y upgrade && \
     apt-get install -y build-essential && \
     apt-get install -y git golang go-dep && \
     rm -rf /var/lib/apt/lists/*
ENV GOPATH /go
ENV CGO_ENABLED 0
COPY ./ ./
RUN dep ensure
RUN go build .

FROM ubuntu:14.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /go/src/github.com/prebid/prebid-server/prebid-server .
COPY static static/
COPY stored_requests/data stored_requests/data
RUN apt-get install -y mtr
EXPOSE 8000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
