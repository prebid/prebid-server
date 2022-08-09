package router

import (
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	originalSchemaDirectory := schemaDirectory
	originalinfoDirectory := infoDirectory
	defer func() {
		schemaDirectory = originalSchemaDirectory
		infoDirectory = originalinfoDirectory
	}()
	schemaDirectory = "../static/bidder-params"
	infoDirectory = "../static/bidder-info"

	type args struct {
		cfg           *config.Configuration
		rateConvertor *currency.RateConverter
	}
	tests := []struct {
		name    string
		args    args
		wantR   *Router
		wantErr bool
		setup   func()
	}{
		{
			name: "Happy path",
			args: args{
				cfg:           &config.Configuration{Adapters: map[string]config.Adapter{"pubmatic": {}}},
				rateConvertor: &currency.RateConverter{},
			},
			wantR:   &Router{Router: &httprouter.Router{}},
			wantErr: false,
			setup: func() {
				g_syncers = nil
				g_cfg = nil
				g_ex = nil
				g_accounts = nil
				g_paramsValidator = nil
				g_storedReqFetcher = nil
				g_storedRespFetcher = nil
				g_metrics = nil
				g_analytics = nil
				g_disabledBidders = nil
				g_videoFetcher = nil
				g_activeBidders = nil
				g_defReqJSON = nil
				g_cacheClient = nil
				g_transport = nil
				g_gdprPermsBuilder = nil
				g_tcf2CfgBuilder = nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			_, err := New(tt.args.cfg, tt.args.rateConvertor)
			assert.Equal(t, tt.wantErr, err != nil, err)

			assert.NotNil(t, g_syncers)
			assert.NotNil(t, g_cfg)
			assert.NotNil(t, g_ex)
			assert.NotNil(t, g_accounts)
			assert.NotNil(t, g_paramsValidator)
			assert.NotNil(t, g_storedReqFetcher)
			assert.NotNil(t, g_storedRespFetcher)
			assert.NotNil(t, g_metrics)
			assert.NotNil(t, g_analytics)
			assert.NotNil(t, g_disabledBidders)
			assert.NotNil(t, g_videoFetcher)
			assert.NotNil(t, g_activeBidders)
			assert.NotNil(t, g_defReqJSON)
			assert.NotNil(t, g_cacheClient)
			assert.NotNil(t, g_transport)
			assert.NotNil(t, g_gdprPermsBuilder)
			assert.NotNil(t, g_tcf2CfgBuilder)
		})
	}
}
