FROM alpine:3.8 AS build
WORKDIR /go/src/github.com/prebid/prebid-server/
RUN apk add -U --no-cache go git dep musl-dev
ENV GOPATH /go
COPY ./ ./
RUN dep ensure
RUN go build .


FROM alpine:3.8 AS release
MAINTAINER Brian O'Kelley <bokelley@appnexus.com>
WORKDIR /usr/local/bin/
COPY --from=build /go/src/github.com/prebid/prebid-server/prebid-server .
COPY static static/
COPY stored_requests/data stored_requests/data
RUN apk add -U --no-cache ca-certificates
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
