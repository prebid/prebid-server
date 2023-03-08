package processor

import (
	"testing"

	"github.com/prebid/prebid-server/config"
)

func Test_emptyProcessor_Replace(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "emtpy processor",
			args: args{
				url: "http://tracker.com?macro1=##PBS_BIDID##&macro2=##PBS_APPBUNDLE##&macro3=##PBS_APPBUNDLE##&macro4=##PBS_PUBDOMAIN##&macro5=##PBS_PAGEURL##&macro6=##PBS_ACCOUNTID##&macro6=##PBS_LIMITADTRACKING##&macro7=##PBS_GDPRCONSENT##&macro8=##PBS_GDPRCONSENT##&macro9=##PBS_MACRO_CUST1##&macro10=##PBS_MACRO_CUST2##",
			},
			want:    "http://tracker.com?macro1=##PBS_BIDID##&macro2=##PBS_APPBUNDLE##&macro3=##PBS_APPBUNDLE##&macro4=##PBS_PUBDOMAIN##&macro5=##PBS_PAGEURL##&macro6=##PBS_ACCOUNTID##&macro6=##PBS_LIMITADTRACKING##&macro7=##PBS_GDPRCONSENT##&macro8=##PBS_GDPRCONSENT##&macro9=##PBS_MACRO_CUST1##&macro10=##PBS_MACRO_CUST2##",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewProcessor(config.MacroProcessorConfig{})
			got, err := e.Replace(tt.args.url, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("emptyProcessor.Replace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("emptyProcessor.Replace() = %v, want %v", got, tt.want)
			}
		})
	}
}
