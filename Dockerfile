FROM billyteves/alpine-golang-glide:1.2.0
MAINTAINER Brian O'Kelley <bokelley@appnexus.com>

RUN mkdir -p /go/src/github.com/prebid/prebid-server
ADD . /go/src/github.com/prebid/prebid-server

WORKDIR /go/src/github.com/prebid/prebid-server
RUN glide install
RUN go build -v .

EXPOSE 8000
ENTRYPOINT ["/go/src/github.com/prebid/prebid-server/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
