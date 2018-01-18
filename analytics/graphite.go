package analytics

type GraphiteLogger struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (g *GraphiteLogger) logAuctionObject(ao *AuctionObject) {
	//Code to parse the object and log in a way required
}

func (g *GraphiteLogger) logSetUIDObject(so *SetUIDObject) {
	//Code to parse the object and log in a way required
}
func (g *GraphiteLogger) logCookieSyncObject(cso *CookieSyncObject) {
	//Code to parse the object and log in a way required
}

func (g *GraphiteLogger) setupGraphiteLogger() *GraphiteLogger {
	//Any other settings can be configured here
	//setupMeters
	return g
}
