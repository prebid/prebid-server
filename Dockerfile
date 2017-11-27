FROM alpine
MAINTAINER Brian O'Kelley <bokelley@appnexus.com>
ADD prebid-server prebid-server
COPY static static/
COPY stored_requests/data stored_requests/data
EXPOSE 8000
ENTRYPOINT ["/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
