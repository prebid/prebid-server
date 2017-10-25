# Deployment

## Packaging

Prebid Server is [packaged with Docker](https://www.docker.com/what-docker) and
optimized to create [lightweight containers](https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/).

[Install Docker](https://www.docker.com/community-edition#/download) and build a container:

```bash
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
docker build -t prebid-server .
```

Test locally with:

```bash
docker run -p 8000:8000 -t prebid-server
```

The server can be reached at `http://localhost:8000`.
