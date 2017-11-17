FROM alpine
MAINTAINER Brian O'Kelley <bokelley@appnexus.com>
ADD prebid-server prebid-server
COPY static static/
COPY openrtb2_configs openrtb2_configs/
EXPOSE 8000
ENTRYPOINT ["/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
