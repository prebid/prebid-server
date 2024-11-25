FROM golang:1.22.3-alpine AS builder
ENV CGO_ENABLED=0
RUN mkdir /build
ADD ./ /build/
WORKDIR /build
RUN env GOOS=linux GOARCH=amd64 go build -o main -a .

FROM alpine
WORKDIR /app/
COPY ./static static/
COPY ./stored_requests/data stored_requests/data
COPY --from=builder /build/main /app/
CMD ["./main"]
