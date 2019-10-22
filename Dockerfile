FROM ubuntu:18.04 AS build
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y wget
RUN cd /tmp && \
    wget https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz && \
    tar -xf go1.12.7.linux-amd64.tar.gz && \
    mv go /usr/local
RUN mkdir -p /app/prebid-server/
WORKDIR /app/prebid-server/
ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
RUN apt-get update && \
    apt-get install -y git && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV CGO_ENABLED 0
COPY ./ ./
RUN go build -mod=vendor .

FROM ubuntu:18.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-server/ .
COPY static static/
COPY stored_requests/data stored_requests/data
RUN apt-get update && \
    apt-get install -y ca-certificates mtr && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
EXPOSE 8000
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
