module github.com/prebid/prebid-server/v3

go 1.23.0

retract v3.0.0 // Forgot to update major version in import path and module name

require (
	github.com/51Degrees/device-detection-go/v4 v4.4.35
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/IABTechLab/adscert v0.34.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/alitto/pond v1.8.3
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/benbjohnson/clock v1.3.0
	github.com/buger/jsonparser v1.1.1
	github.com/chasex/glog v0.0.0-20160217080310-c62392af379c
	github.com/coocood/freecache v1.2.1
	github.com/docker/go-units v0.4.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/golang/glog v1.2.4
	github.com/google/go-cmp v0.6.0
	github.com/json-iterator/go v1.1.12
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lib/pq v1.10.4
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/modern-go/reflect2 v1.0.2
	github.com/prebid/go-gdpr v1.12.0
	github.com/prebid/go-gpp v0.2.0
	github.com/prebid/openrtb/v20 v20.3.0
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.2.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/rs/cors v1.11.0
	github.com/spf13/cast v1.5.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.1
	github.com/tidwall/gjson v1.17.1
	github.com/tidwall/sjson v1.2.5
	github.com/vrischmann/go-metrics-influxdb v0.1.1
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/yudai/gojsondiff v1.0.0
	golang.org/x/net v0.38.0
	golang.org/x/text v0.23.0
	google.golang.org/grpc v1.56.3
	gopkg.in/evanphx/json-patch.v5 v5.9.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
