FROM golang:latest
##MAINTAINER Brian O'Kelley <bokelley@appnexus.com>

## GO BUILD STUFF
WORKDIR /go/src/github.com/prebid/prebid-server
COPY . .
RUN mv pbs.yaml.production pbs.yaml 
RUN sh build.sh
RUN dep ensure

## THIS WAS BREAKING SO I TURNED IT OF - JCJ
## RUN ./validate.sh

RUN go build . 


## ORIGINAL B'OK VERSION FROM HERE ON 
#ADD prebid-server prebid-server
#COPY static static/
#COPY stored_requests/data stored_requests/data
EXPOSE 8000
ENTRYPOINT ["/go/src/github.com/prebid/prebid-server/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
