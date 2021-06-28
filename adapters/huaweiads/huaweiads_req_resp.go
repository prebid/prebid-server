package huaweiads

type HuaweiAdsRequest struct {
	Version           string     `json:"version"`
	Multislot         []Adslot30 `json:"multislot"`
	App               App        `json:"app"`
	Device            Device     `json:"device"`
	Network           Network    `json:"network,omitempty"`
	ClientAdRequestId string     `json:"clientAdRequestId,omitempty"`
	ParentCtrlUser    int32      `json:"parentCtrlUser,omitempty"`
	NonPersonalizedAd int32      `json:"nonPersonalizedAd,omitempty"`
	Regs              Regs       `json:"regs,omitempty"`
	Geo               Geo        `json:"geo,omitempty"`
	Consent           string     `json:"consent,omitempty"`
}

type Adslot30 struct {
	Slotid        string `json:"slotid"`
	Adtype        int32  `json:"adtype"`
	Test          int32  `json:"test"`
	TotalDuration int32  `json:"totalDuration,omitempty"`
	Orientation   int32  `json:"orientation,omitempty"`
}

type App struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
	Pkgname string `json:"pkgname"`
	Lang    string `json:"lang,omitempty"`
	Country string `json:"country,omitempty"`
}

type Device struct {
	Type                int32   `json:"type,omitempty"`
	Useragent           string  `json:"useragent,omitempty"`
	Os                  string  `json:"os,omitempty"`
	Version             string  `json:"version,omitempty"`
	Maker               string  `json:"maker,omitempty"`
	Model               string  `json:"model,omitempty"`
	Width               int32   `json:"width,omitempty"`
	Height              int32   `json:"height,omitempty"`
	Language            string  `json:"language,omitempty"`
	BuildVersion        string  `json:"buildVersion,omitempty"`
	Dpi                 int32   `json:"dpi,omitempty"`
	Pxratio             float32 `json:"pxratio,omitempty"`
	Imei                string  `json:"imei,omitempty"`
	Oaid                string  `json:"oaid,omitempty"`
	IsTrackingEnabled   string  `json:"isTrackingEnabled,omitempty"`
	EmuiVer             string  `json:"emuiVer,omitempty"`
	LocaleCountry       string  `json:"localeCountry"`
	SimCountryIso       string  `json:"simCountryIso,omitempty"`
	BelongCountry       string  `json:"belongCountry"`
	GaidTrackingEnabled string  `json:"gaidTrackingEnabled,omitempty"`
	Gaid                string  `json:"gaid,omitempty"`
	VerCodeOfHms        string  `json:"verCodeOfHms,omitempty"`
	ClientTime          string  `json:"clientTime"`
	VerCodeOfAG         string  `json:"verCodeOfAG,omitempty"`
	VendorCountry       string  `json:"vendorCountry,omitempty"`
	RoLocaleCountry     string  `json:"roLocaleCountry,omitempty"`
	AgCountryCode       string  `json:"agCountryCode,omitempty"`
	RouterCountry       string  `json:"routerCountry,omitempty"`
	RoLocale            string  `json:"roLocale,omitempty"`
	Ip                  string  `json:"ip,omitempty"`
}

type Network struct {
	Type     int32      `json:"type"`
	Carrier  int32      `json:"carrier,omitempty"`
	CellInfo []CellInfo `json:"cellInfo,omitempty"`
}

type Regs struct {
	Coppa    int32  `json:"coppa,omitempty"`
	Tfua     int32  `json:"tfua,omitempty"`
	AdRating string `json:"adRating,omitempty"`
}

type Geo struct {
	Lon      float32 `json:"lon,omitempty"`
	Lat      float32 `json:"lat,omitempty"`
	Accuracy int32   `json:"accuracy,omitempty"`
	Lastfix  int32   `json:"lastfix,omitempty"`
}

type CellInfo struct {
	Mcc string `json:"mcc,omitempty"`
	Mnc string `json:"mnc,omitempty"`
}

type HuaweiAdsResponse struct {
	Retcode int32  `json:"retcode"`
	Reason  string `json:"reason"`
	Multiad []Ad30 `json:"multiad"`
}

type Ad30 struct {
	AdType    int32     `json:"adtype"`
	Slotid    string    `json:"slotid"`
	Retcode30 int32     `json:"retcode30"`
	Content   []Content `json:"content"`
}

type Content struct {
	Contentid       string     `json:"contentid"`
	Endtime         int64      `json:"endtime"`
	Interactiontype int32      `json:"interactiontype"`
	Creativetype    int32      `json:"creativetype"`
	MetaData        MetaData   `json:"metaData"`
	Starttime       int64      `json:"starttime"`
	KeyWords        []string   `json:"keyWords"`
	KeyWordsType    []string   `json:"keyWordsType"`
	Monitor         []Monitor  `json:"monitor"`
	RewardItem      RewardItem `json:"rewardItem"`
	WhyThisAd       string     `json:"whyThisAd"`
	Cur             string     `json:"cur"`
	Price           float64    `json:"price"`
}

type MetaData struct {
	Title             string      `json:"title"`
	ImageInfo         []ImageInfo `json:"imageInfo"`
	Icon              []Icon      `json:"icon"`
	ClickUrl          string      `json:"clickUrl"`
	Label             string      `json:"label"`
	Intent            string      `json:"intent"`
	VideoInfo         VideoInfo   `json:"videoInfo"`
	ApkInfo           ApkInfo     `json:"apkInfo"`
	Duration          int64       `json:"duration"`
	MediaFile         MediaFile   `json:"mediaFile"`
	RewardCriterion   string      `json:"rewardCriterion"`
	ScreenOrientation string      `json:"screenOrientation"`
	PrivacyUrl        string      `json:"privacyUrl"`
}

type ImageInfo struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type Icon struct {
	Url       string `json:"url"`
	Height    int64  `json:"height"`
	FileSize  int64  `json:"fileSize"`
	Sha256    string `json:"sha256"`
	ImageType string `json:"imageType"`
	Width     int64  `json:"width"`
}

type VideoInfo struct {
	VideoDownloadUrl string  `json:"videoDownloadUrl"`
	VideoDuration    int32   `json:"videoDuration"`
	VideoFileSize    int32   `json:"videoFileSize"`
	Sha256           string  `json:"sha256"`
	VideoRatio       float32 `json:"videoRatio"`
}

type ApkInfo struct {
	Url           string       `json:"url"`
	FileSize      int64        `json:"fileSize"`
	Sha256        string       `json:"sha256"`
	PackageName   string       `json:"packageName"`
	SecondUrl     string       `json:"secondUrl"`
	AppName       string       `json:"appName"`
	Permissions   []Permission `json:"permissions"`
	VersionName   string       `json:"versionName"`
	AppDesc       string       `json:"appDesc"`
	AppIcon       string       `json:"appIcon"`
	DeveloperName string       `json:"developerName"`
}

type MediaFile struct {
	Mime     string `json:"mime"`
	Width    int64  `json:"width"`
	Height   int64  `json:"height"`
	FileSize int64  `json:"fileSize"`
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
}

type Monitor struct {
	EventType string   `json:"eventType"`
	Url       []string `json:"url"`
}

type Permission struct {
	PermissionLabel string `json:"permissionLabel"`
	GroupDesc       string `json:"groupDesc"`
	TargetSDK       string `json:"targetSDK"`
}

type RewardItem struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}
