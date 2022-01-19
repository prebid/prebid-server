module github.com/PubMatic-OpenWrap/prebid-server

go 1.16

require (
	github.com/DATA-DOG/go-sqlmock v1.3.0
	github.com/NYTimes/gziphandler v1.1.1
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	github.com/beevik/etree v1.0.2
	github.com/buger/jsonparser v1.1.1
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/chasex/glog v0.0.0-20160217080310-c62392af379c
	github.com/coocood/freecache v1.0.1
	github.com/docker/go-units v0.4.0
	github.com/evanphx/json-patch v0.0.0-20180720181644-f195058310bd
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/influxdata/influxdb v1.6.1
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lib/pq v1.0.0
	github.com/magiconair/properties v1.8.5
	github.com/mitchellh/copystructure v1.1.2
	github.com/mxmCherry/openrtb v13.0.0+incompatible
	github.com/mxmCherry/openrtb/v15 v15.0.0
	github.com/prebid/go-gdpr v1.11.0
	github.com/prebid/prebid-server v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/rcrowley/go-metrics v0.0.0-20180503174638-e2704e165165
	github.com/rs/cors v1.5.0
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/vrischmann/go-metrics-influxdb v0.0.0-20160917065939-43af8332c303
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v0.0.0-20180816142147-da425ebb7609
	github.com/yudai/gojsondiff v0.0.0-20170107030110-7b1b7adf999d
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yudai/pp v2.0.1+incompatible // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/text v0.3.6
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/prebid/prebid-server => ./

replace github.com/mxmCherry/openrtb/v15 v15.0.0 => github.com/PubMatic-OpenWrap/openrtb/v15 v15.0.0-20210425063110-b01110089669

replace github.com/beevik/etree v1.0.2 => github.com/PubMatic-OpenWrap/etree v1.0.2-0.20210129100623-8f30cfecf9f4
