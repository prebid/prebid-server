package agma

type Config struct {
	Enabled  bool            `json:"enabled"`
	Endpoint EndpointConfig  `json:"endpoint"`
	Buffers  BufferConfig    `json:"buffers"`
	Accounts []AccountConfig `json:"accounts"`
}

type EndpointConfig struct {
	Url     string `json:"url"`
	Timeout string `json:"timeout"`
	Gzip    bool   `json:"gzip"`
}

type BufferConfig struct {
	BufferSize string `json:"size"`
	EventCount int    `json:"count"`
	Timeout    string `json:"timeout"`
}

type AccountConfig struct {
	Code        string `json:"code"`
	PublisherId string `json:"publisher_id"`
	SiteAppId   string `json:"site_app_id"`
}
