package floors

type FloorsConfig map[string]SiteConfig

type SiteConfig struct {
	siteID    string
	siteUID   string
	platforms []string
	countries []string
}

var floorsConfig = FloorsConfig{
	"ViXOj3": SiteConfig{
		siteID:  "1992",
		siteUID: "ViXOj3",
		platforms: []string{"m-android|chrome",
			"m-ios|chrome",
			"t-android|chrome",
			"t-ios|chrome",
			"w|chrome",
			"m-android|edge",
			"t-android|edge",
			"t-ios|edge",
			"w|edge",
			"m-ios|google search",
			"t-ios|google search",
			"t-ios|safari",
			"m-ios|safari",
			"w|safari"},
		countries: []string{"US"},
	},
}
