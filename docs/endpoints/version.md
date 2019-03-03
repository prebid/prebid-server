## `GET /version`

This endpoint exposes the application version as defined at compilation time.
Version can be set either:
- manually using go build -ldflags "-X main.Rev=`git rev-parse --short HEAD`"
- automatically via .travis.yml configuration
See section in (../../pbs_light.go):

```go
// Holds binary revision string
// Set manually at build time using:
//    go build -ldflags "-X main.Rev=`git rev-parse --short HEAD`"
// Populated automatically at build / release time via .travis.yml
//   `gox -os="linux" -arch="386" -output="{{.Dir}}_{{.OS}}_{{.Arch}}" -ldflags "-X main.Rev=`git rev-parse --short HEAD`" -verbose ./...;`
// See issue #559
var Rev string
```

### Sample responses

#### Version set
```json
{"revision": "d6cd1e2bd19e03a81132a23b2025920577f84e37"},
```

#### Version not set
```json
{"revision": "not-set"}`
```
