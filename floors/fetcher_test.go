package floors

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/alitto/pond"
	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConf "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
	"github.com/stretchr/testify/assert"
)

const MaxAge = "max-age"

func TestFetchQueueLen(t *testing.T) {
	tests := []struct {
		name string
		fq   FetchQueue
		want int
	}{
		{
			name: "Queue is empty",
			fq:   make(FetchQueue, 0),
			want: 0,
		},
		{
			name: "Queue is of lenght 1",
			fq:   make(FetchQueue, 1),
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fq.Len(); got != tt.want {
				t.Errorf("FetchQueue.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchQueueLess(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		fq   FetchQueue
		args args
		want bool
	}{
		{
			name: "first fetchperiod is less than second",
			fq:   FetchQueue{&fetchInfo{fetchTime: 10}, &fetchInfo{fetchTime: 20}},
			args: args{i: 0, j: 1},
			want: true,
		},
		{
			name: "first fetchperiod is greater than second",
			fq:   FetchQueue{&fetchInfo{fetchTime: 30}, &fetchInfo{fetchTime: 10}},
			args: args{i: 0, j: 1},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fq.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("FetchQueue.Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchQueueSwap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		fq   FetchQueue
		args args
	}{
		{
			name: "Swap two elements at index i and j",
			fq:   FetchQueue{&fetchInfo{fetchTime: 30}, &fetchInfo{fetchTime: 10}},
			args: args{i: 0, j: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fInfo1, fInfo2 := tt.fq[0], tt.fq[1]
			tt.fq.Swap(tt.args.i, tt.args.j)
			assert.Equal(t, fInfo1, tt.fq[1], "elements are not swapped")
			assert.Equal(t, fInfo2, tt.fq[0], "elements are not swapped")
		})
	}
}

func TestFetchQueuePush(t *testing.T) {
	type args struct {
		element interface{}
	}
	tests := []struct {
		name string
		fq   *FetchQueue
		args args
	}{
		{
			name: "Push element to queue",
			fq:   &FetchQueue{},
			args: args{element: &fetchInfo{fetchTime: 10}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fq.Push(tt.args.element)
			q := *tt.fq
			assert.Equal(t, q[0], &fetchInfo{fetchTime: 10})
		})
	}
}

func TestFetchQueuePop(t *testing.T) {
	tests := []struct {
		name string
		fq   *FetchQueue
		want interface{}
	}{
		{
			name: "Pop element from queue",
			fq:   &FetchQueue{&fetchInfo{fetchTime: 10}},
			want: &fetchInfo{fetchTime: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fq.Pop(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchQueue.Pop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchQueueTop(t *testing.T) {
	tests := []struct {
		name string
		fq   *FetchQueue
		want *fetchInfo
	}{
		{
			name: "Get top element from queue",
			fq:   &FetchQueue{&fetchInfo{fetchTime: 20}},
			want: &fetchInfo{fetchTime: 20},
		},
		{
			name: "Queue is empty",
			fq:   &FetchQueue{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fq.Top(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchQueue.Top() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePriceFloorRules(t *testing.T) {
	var zero = 0
	var one_o_one = 101
	var testURL = "abc.com"
	type args struct {
		configs     config.AccountFloorFetch
		priceFloors *openrtb_ext.PriceFloorRules
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Price floor data is empty",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      5,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{},
			},
			wantErr: true,
		},
		{
			name: "Model group array is empty",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      5,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{},
				},
			},
			wantErr: true,
		},
		{
			name: "floor rules is empty",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      5,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{},
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "floor rules is grater than max floor rules",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      0,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Modelweight is zero",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
							ModelWeight: &zero,
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Modelweight is 101",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
							ModelWeight: &one_o_one,
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "skiprate is 101",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
							SkipRate: 101,
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Default is -1",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
							Default: -1,
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid skip rate in data",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						SkipRate: -44,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
						}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid UseFetchDataRate",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					URL:           testURL,
					Timeout:       5,
					MaxFileSizeKB: 20,
					MaxRules:      1,
					MaxAge:        20,
					Period:        10,
				},
				priceFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						SkipRate: 10,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
							Values: map[string]float64{
								"*|*|www.website.com": 15.01,
							},
						}},
						UseFetchDataRate: ptrutil.ToPtr(-11),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRules(tt.args.configs, tt.args.priceFloors); (err != nil) != tt.wantErr {
				t.Errorf("validatePriceFloorRules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchFloorRulesFromURL(t *testing.T) {
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Add("Content-Length", "645")
			w.Header().Add(MaxAge, "20")
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	type args struct {
		configs config.AccountFloorFetch
	}
	tests := []struct {
		name           string
		args           args
		response       []byte
		responseStatus int
		want           []byte
		want1          int
		wantErr        bool
	}{
		{
			name: "Floor data is successfully returned",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "",
					Timeout: 60,
					Period:  300,
				},
			},
			response: func() []byte {
				data := `{"data":{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want: func() []byte {
				data := `{"data":{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
				return []byte(data)
			}(),
			want1:   20,
			wantErr: false,
		},
		{
			name: "Time out occured",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "",
					Timeout: 0,
					Period:  300,
				},
			},
			want1:          0,
			responseStatus: 200,
			wantErr:        true,
		},
		{
			name: "Invalid URL",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "%%",
					Timeout: 10,
					Period:  300,
				},
			},
			want1:          0,
			responseStatus: 200,
			wantErr:        true,
		},
		{
			name: "No response from server",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "",
					Timeout: 10,
					Period:  300,
				},
			},
			want1:          0,
			responseStatus: 500,
			wantErr:        true,
		},
		{
			name: "Invalid response",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "",
					Timeout: 10,
					Period:  300,
				},
			},
			want1:          0,
			response:       []byte("1"),
			responseStatus: 200,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHttpServer := httptest.NewServer(mockHandler(tt.response, tt.responseStatus))
			defer mockHttpServer.Close()

			if tt.args.configs.URL == "" {
				tt.args.configs.URL = mockHttpServer.URL
			}
			pff := PriceFloorFetcher{
				httpClient: mockHttpServer.Client(),
			}
			got, got1, err := pff.fetchFloorRulesFromURL(tt.args.configs)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchFloorRulesFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchFloorRulesFromURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fetchFloorRulesFromURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFetchFloorRulesFromURLInvalidMaxAge(t *testing.T) {
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Add("Content-Length", "645")
			w.Header().Add(MaxAge, "abc")
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	type args struct {
		configs config.AccountFloorFetch
	}
	tests := []struct {
		name           string
		args           args
		response       []byte
		responseStatus int
		want           []byte
		want1          int
		wantErr        bool
	}{
		{
			name: "Floor data is successfully returned",
			args: args{
				configs: config.AccountFloorFetch{
					URL:     "",
					Timeout: 60,
					Period:  300,
				},
			},
			response: func() []byte {
				data := `{"data":{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want: func() []byte {
				data := `{"data":{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
				return []byte(data)
			}(),
			want1:   0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHttpServer := httptest.NewServer(mockHandler(tt.response, tt.responseStatus))
			defer mockHttpServer.Close()

			if tt.args.configs.URL == "" {
				tt.args.configs.URL = mockHttpServer.URL
			}

			ppf := PriceFloorFetcher{
				httpClient: mockHttpServer.Client(),
			}
			got, got1, err := ppf.fetchFloorRulesFromURL(tt.args.configs)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchFloorRulesFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchFloorRulesFromURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fetchFloorRulesFromURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFetchAndValidate(t *testing.T) {
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Add(MaxAge, "30")
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	type args struct {
		configs config.AccountFloorFetch
	}
	tests := []struct {
		name           string
		args           args
		response       []byte
		responseStatus int
		want           *openrtb_ext.PriceFloorRules
		want1          int
	}{
		{
			name: "Recieved valid price floor rules response",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					Timeout:       30,
					MaxFileSizeKB: 700,
					MaxRules:      30,
					MaxAge:        60,
					Period:        40,
				},
			},
			response: func() []byte {
				data := `{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want: func() *openrtb_ext.PriceFloorRules {
				var res openrtb_ext.PriceFloorRules
				data := `{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`
				_ = json.Unmarshal([]byte(data), &res.Data)
				return &res
			}(),
			want1: 30,
		},
		{
			name: "No response from server",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					Timeout:       30,
					MaxFileSizeKB: 700,
					MaxRules:      30,
					MaxAge:        60,
					Period:        40,
				},
			},
			response:       []byte{},
			responseStatus: 500,
			want:           nil,
			want1:          0,
		},
		{
			name: "File is greater than MaxFileSize",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					Timeout:       30,
					MaxFileSizeKB: 1,
					MaxRules:      30,
					MaxAge:        60,
					Period:        40,
				},
			},
			response: func() []byte {
				data := `{"currency":"USD","floorProvider":"PM","floorsSchemaVersion":2,"modelGroups":[{"modelVersion":"M_0","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":0.85,"www.missyusa.com":0.7}},{"modelVersion":"M_1","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":1,"www.missyusa.com":1.85}},{"modelVersion":"M_2","modelWeight":5,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.6,"www.missyusa.com":0.7}},{"modelVersion":"M_3","modelWeight":2,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.9,"www.missyusa.com":0.75}},{"modelVersion":"M_4","modelWeight":1,"schema":{"fields":["domain"]},"values":{"www.missyusa.com":1.35,"missyusa.com":1.75}},{"modelVersion":"M_5","modelWeight":2,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.4,"www.missyusa.com":0.9}},{"modelVersion":"M_6","modelWeight":43,"schema":{"fields":["domain"]},"values":{"www.missyusa.com":2,"missyusa.com":2}},{"modelVersion":"M_7","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.4,"www.missyusa.com":1.85}},{"modelVersion":"M_8","modelWeight":3,"schema":{"fields":["domain"]},"values":{"www.missyusa.com":1.7,"missyusa.com":0.1}},{"modelVersion":"M_9","modelWeight":7,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.9,"www.missyusa.com":1.05}},{"modelVersion":"M_10","modelWeight":9,"schema":{"fields":["domain"]},"values":{"www.missyusa.com":2,"missyusa.com":0.1}},{"modelVersion":"M_11","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":0.45,"www.missyusa.com":1.5}},{"modelVersion":"M_12","modelWeight":8,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.2,"www.missyusa.com":1.7}},{"modelVersion":"M_13","modelWeight":8,"schema":{"fields":["domain"]},"values":{"missyusa.com":0.85,"www.missyusa.com":0.75}},{"modelVersion":"M_14","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.8,"www.missyusa.com":1}},{"modelVersion":"M_15","modelWeight":1,"schema":{"fields":["domain"]},"values":{"www.missyusa.com":1.2,"missyusa.com":1.75}},{"modelVersion":"M_16","modelWeight":2,"schema":{"fields":["domain"]},"values":{"missyusa.com":1,"www.missyusa.com":0.7}},{"modelVersion":"M_17","modelWeight":1,"schema":{"fields":["domain"]},"values":{"missyusa.com":0.45,"www.missyusa.com":0.35}},{"modelVersion":"M_18","modelWeight":3,"schema":{"fields":["domain"]},"values":{"missyusa.com":1.2,"www.missyusa.com":1.05}}],"skipRate":10}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want:           nil,
			want1:          0,
		},
		{
			name: "Malformed response : json unmarshalling failed",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					Timeout:       30,
					MaxFileSizeKB: 800,
					MaxRules:      30,
					MaxAge:        60,
					Period:        40,
				},
			},
			response: func() []byte {
				data := `{"data":nil?}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want:           nil,
			want1:          0,
		},
		{
			name: "Validations failed for price floor rules response",
			args: args{
				configs: config.AccountFloorFetch{
					Enabled:       true,
					Timeout:       30,
					MaxFileSizeKB: 700,
					MaxRules:      30,
					MaxAge:        60,
					Period:        40,
				},
			},
			response: func() []byte {
				data := `{"data":{"currency":"USD","modelgroups":[]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
				return []byte(data)
			}(),
			responseStatus: 200,
			want:           nil,
			want1:          0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHttpServer := httptest.NewServer(mockHandler(tt.response, tt.responseStatus))
			defer mockHttpServer.Close()
			ppf := PriceFloorFetcher{
				httpClient: mockHttpServer.Client(),
			}
			tt.args.configs.URL = mockHttpServer.URL
			got, got1 := ppf.fetchAndValidate(tt.args.configs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchAndValidate() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fetchAndValidate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func mockFetcherInstance(config config.PriceFloors, httpClient *http.Client, metricEngine metrics.MetricsEngine) *PriceFloorFetcher {
	if !config.Enabled {
		return nil
	}

	floorFetcher := PriceFloorFetcher{
		pool:            pond.New(config.Fetcher.Worker, config.Fetcher.Capacity, pond.PanicHandler(workerPanicHandler)),
		fetchQueue:      make(FetchQueue, 0, 100),
		fetchInProgress: make(map[string]bool),
		configReceiver:  make(chan fetchInfo, config.Fetcher.Capacity),
		done:            make(chan struct{}),
		cache:           freecache.NewCache(config.Fetcher.CacheSize * 1024 * 1024),
		httpClient:      httpClient,
		time:            &timeutil.RealTime{},
		metricEngine:    metricEngine,
		maxRetries:      10,
	}

	go floorFetcher.Fetcher()

	return &floorFetcher
}

func TestFetcherWhenRequestGetSameURLInrequest(t *testing.T) {
	refetchCheckInterval = 1
	response := []byte(`{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`)
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(response, 200))
	defer mockHttpServer.Close()

	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize: 1,
			Worker:    5,
			Capacity:  10,
		},
	}
	fetcherInstance := mockFetcherInstance(floorConfig, mockHttpServer.Client(), &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        1,
		},
	}

	for i := 0; i < 50; i++ {
		fetcherInstance.Fetch(fetchConfig)
	}

	assert.Never(t, func() bool { return len(fetcherInstance.fetchQueue) > 1 }, time.Duration(2*time.Second), 100*time.Millisecond, "Queue Got more than one entry")
	assert.Never(t, func() bool { return len(fetcherInstance.fetchInProgress) > 1 }, time.Duration(2*time.Second), 100*time.Millisecond, "Map Got more than one entry")

}

func TestFetcherDataPresentInCache(t *testing.T) {
	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize: 1,
			Worker:    2,
			Capacity:  5,
		},
	}

	fetcherInstance := mockFetcherInstance(floorConfig, http.DefaultClient, &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com/floor",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        5,
		},
	}
	var res *openrtb_ext.PriceFloorRules
	data := `{"data":{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"floormin":1,"enforcement":{"enforcepbs":false,"floordeals":true}}`
	_ = json.Unmarshal([]byte(data), &res)
	fetcherInstance.SetWithExpiry("http://test.com/floor", []byte(data), fetchConfig.Fetcher.MaxAge)

	val, status := fetcherInstance.Fetch(fetchConfig)
	assert.Equal(t, res, val, "Invalid value in cache or cache is empty")
	assert.Equal(t, "success", status, "Floor fetch should be success")
}

func TestFetcherDataNotPresentInCache(t *testing.T) {
	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize: 1,
			Worker:    2,
			Capacity:  5,
		},
	}

	fetcherInstance := mockFetcherInstance(floorConfig, http.DefaultClient, &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com/floor",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        5,
		},
	}
	fetcherInstance.SetWithExpiry("http://test.com/floor", nil, fetchConfig.Fetcher.MaxAge)

	val, status := fetcherInstance.Fetch(fetchConfig)

	assert.Equal(t, (*openrtb_ext.PriceFloorRules)(nil), val, "Floor data should be nil")
	assert.Equal(t, "error", status, "Floor fetch should be error")
}

func TestFetcherEntryNotPresentInCache(t *testing.T) {
	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize:  1,
			Worker:     2,
			Capacity:   5,
			MaxRetries: 10,
		},
	}

	fetcherInstance := NewPriceFloorFetcher(floorConfig, http.DefaultClient, &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com/floor",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        5,
		},
	}

	val, status := fetcherInstance.Fetch(fetchConfig)

	assert.Equal(t, (*openrtb_ext.PriceFloorRules)(nil), val, "Floor data should be nil")
	assert.Equal(t, openrtb_ext.FetchInprogress, status, "Floor fetch should be error")
}

func TestFetcherDynamicFetchDisable(t *testing.T) {
	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize:  1,
			Worker:     2,
			Capacity:   5,
			MaxRetries: 5,
		},
	}

	fetcherInstance := NewPriceFloorFetcher(floorConfig, http.DefaultClient, &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: false,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com/floor",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        5,
		},
	}

	val, status := fetcherInstance.Fetch(fetchConfig)

	assert.Equal(t, (*openrtb_ext.PriceFloorRules)(nil), val, "Floor data should be nil")
	assert.Equal(t, openrtb_ext.FetchNone, status, "Floor fetch should be error")
}

func TestPriceFloorFetcherWorker(t *testing.T) {
	var floorData openrtb_ext.PriceFloorData
	response := []byte(`{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`)
	_ = json.Unmarshal(response, &floorData)
	floorResp := &openrtb_ext.PriceFloorRules{
		Data: &floorData,
	}

	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Add(MaxAge, "5")
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(response, 200))
	defer mockHttpServer.Close()

	fetcherInstance := PriceFloorFetcher{
		pool:            nil,
		fetchQueue:      nil,
		fetchInProgress: nil,
		configReceiver:  make(chan fetchInfo, 1),
		done:            nil,
		cache:           freecache.NewCache(1 * 1024 * 1024),
		httpClient:      mockHttpServer.Client(),
		time:            &timeutil.RealTime{},
		metricEngine:    &metricsConf.NilMetricsEngine{},
		maxRetries:      10,
	}
	defer close(fetcherInstance.configReceiver)

	fetchConfig := fetchInfo{
		AccountFloorFetch: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        1,
		},
	}

	fetcherInstance.worker(fetchConfig)
	dataInCache, _ := fetcherInstance.Get(mockHttpServer.URL)
	var gotFloorData *openrtb_ext.PriceFloorRules
	json.Unmarshal(dataInCache, &gotFloorData)
	assert.Equal(t, floorResp, gotFloorData, "Data should be stored in cache")

	info := <-fetcherInstance.configReceiver
	assert.Equal(t, true, info.refetchRequest, "Recieved request is not refetch request")
	assert.Equal(t, mockHttpServer.URL, info.AccountFloorFetch.URL, "Recieved request with different url")

}

func TestPriceFloorFetcherWorkerRetry(t *testing.T) {
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(nil, 500))
	defer mockHttpServer.Close()

	fetcherInstance := PriceFloorFetcher{
		pool:            nil,
		fetchQueue:      nil,
		fetchInProgress: nil,
		configReceiver:  make(chan fetchInfo, 1),
		done:            nil,
		cache:           nil,
		httpClient:      mockHttpServer.Client(),
		time:            &timeutil.RealTime{},
		metricEngine:    &metricsConf.NilMetricsEngine{},
		maxRetries:      5,
	}
	defer close(fetcherInstance.configReceiver)

	fetchConfig := fetchInfo{
		AccountFloorFetch: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        1,
		},
	}

	fetcherInstance.worker(fetchConfig)

	info := <-fetcherInstance.configReceiver
	assert.Equal(t, 1, info.retryCount, "Retry Count is not 1")
}

func TestPriceFloorFetcherWorkerDefaultCacheExpiry(t *testing.T) {
	var floorData openrtb_ext.PriceFloorData
	response := []byte(`{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`)
	_ = json.Unmarshal(response, &floorData)
	floorResp := &openrtb_ext.PriceFloorRules{
		Data: &floorData,
	}

	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(response, 200))
	defer mockHttpServer.Close()

	fetcherInstance := &PriceFloorFetcher{
		pool:            nil,
		fetchQueue:      nil,
		fetchInProgress: nil,
		configReceiver:  make(chan fetchInfo, 1),
		done:            nil,
		cache:           freecache.NewCache(1 * 1024 * 1024),
		httpClient:      mockHttpServer.Client(),
		time:            &timeutil.RealTime{},
		metricEngine:    &metricsConf.NilMetricsEngine{},
		maxRetries:      5,
	}

	fetchConfig := fetchInfo{
		AccountFloorFetch: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        20,
			Period:        1,
		},
	}

	fetcherInstance.worker(fetchConfig)
	dataInCache, _ := fetcherInstance.Get(mockHttpServer.URL)
	var gotFloorData *openrtb_ext.PriceFloorRules
	json.Unmarshal(dataInCache, &gotFloorData)
	assert.Equal(t, floorResp, gotFloorData, "Data should be stored in cache")

	info := <-fetcherInstance.configReceiver
	defer close(fetcherInstance.configReceiver)
	assert.Equal(t, true, info.refetchRequest, "Recieved request is not refetch request")
	assert.Equal(t, mockHttpServer.URL, info.AccountFloorFetch.URL, "Recieved request with different url")

}

func TestPriceFloorFetcherSubmit(t *testing.T) {
	response := []byte(`{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`)
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(response, 200))
	defer mockHttpServer.Close()

	fetcherInstance := &PriceFloorFetcher{
		pool:            pond.New(1, 1),
		fetchQueue:      make(FetchQueue, 0),
		fetchInProgress: nil,
		configReceiver:  make(chan fetchInfo, 1),
		done:            make(chan struct{}),
		cache:           freecache.NewCache(1 * 1024 * 1024),
		httpClient:      mockHttpServer.Client(),
		time:            &timeutil.RealTime{},
		metricEngine:    &metricsConf.NilMetricsEngine{},
		maxRetries:      5,
	}
	defer fetcherInstance.Stop()

	fetchInfo := fetchInfo{
		refetchRequest: false,
		fetchTime:      time.Now().Unix(),
		AccountFloorFetch: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        2,
			Period:        1,
		},
	}

	fetcherInstance.submit(&fetchInfo)

	info := <-fetcherInstance.configReceiver
	assert.Equal(t, true, info.refetchRequest, "Recieved request is not refetch request")
	assert.Equal(t, mockHttpServer.URL, info.AccountFloorFetch.URL, "Recieved request with different url")

}

type testPool struct{}

func (t *testPool) TrySubmit(task func()) bool {
	return false
}

func (t *testPool) Stop() {}

func TestPriceFloorFetcherSubmitFailed(t *testing.T) {
	fetcherInstance := &PriceFloorFetcher{
		pool:            &testPool{},
		fetchQueue:      make(FetchQueue, 0),
		fetchInProgress: nil,
		configReceiver:  nil,
		done:            nil,
		cache:           nil,
	}
	defer fetcherInstance.pool.Stop()

	fetchInfo := fetchInfo{
		refetchRequest: false,
		fetchTime:      time.Now().Unix(),
		AccountFloorFetch: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        2,
			Period:        1,
		},
	}

	fetcherInstance.submit(&fetchInfo)
	assert.Equal(t, 1, len(fetcherInstance.fetchQueue), "Unable to submit the task")
}

func getRandomNumber() int {
	//nolint: staticcheck // SA1019: rand.Seed has been deprecated since Go 1.20 and an alternative has been available since Go 1.0: As of Go 1.20 there is no reason to call Seed with a random value.
	rand.Seed(time.Now().UnixNano())
	min := 1
	max := 10
	return rand.Intn(max-min+1) + min
}

func TestFetcherWhenRequestGetDifferentURLInrequest(t *testing.T) {
	refetchCheckInterval = 1
	response := []byte(`{"currency":"USD","modelgroups":[{"modelweight":40,"modelversion":"version1","default":5,"values":{"banner|300x600|www.website.com":3,"banner|728x90|www.website.com":5,"banner|300x600|*":4,"banner|300x250|*":2,"*|*|*":16,"*|300x250|*":10,"*|300x600|*":12,"*|300x600|www.website.com":11,"banner|*|*":8,"banner|300x250|www.website.com":1,"*|728x90|www.website.com":13,"*|300x250|www.website.com":9,"*|728x90|*":14,"banner|728x90|*":6,"banner|*|www.website.com":7,"*|*|www.website.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]}`)
	mockHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(mockStatus)
			w.Write(mockResponse)
		})
	}

	mockHttpServer := httptest.NewServer(mockHandler(response, 200))
	defer mockHttpServer.Close()

	floorConfig := config.PriceFloors{
		Enabled: true,
		Fetcher: config.PriceFloorFetcher{
			CacheSize:  1,
			Worker:     5,
			Capacity:   10,
			MaxRetries: 5,
		},
	}
	fetcherInstance := mockFetcherInstance(floorConfig, mockHttpServer.Client(), &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           mockHttpServer.URL,
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        5,
			Period:        1,
		},
	}

	for i := 0; i < 50; i++ {
		fetchConfig.Fetcher.URL = fmt.Sprintf("%s?id=%d", mockHttpServer.URL, getRandomNumber())
		fetcherInstance.Fetch(fetchConfig)
	}

	assert.Never(t, func() bool { return len(fetcherInstance.fetchQueue) > 10 }, time.Duration(2*time.Second), 100*time.Millisecond, "Queue Got more than one entry")
	assert.Never(t, func() bool { return len(fetcherInstance.fetchInProgress) > 10 }, time.Duration(2*time.Second), 100*time.Millisecond, "Map Got more than one entry")
}

func TestFetchWhenPriceFloorsDisabled(t *testing.T) {
	floorConfig := config.PriceFloors{
		Enabled: false,
		Fetcher: config.PriceFloorFetcher{
			CacheSize: 1,
			Worker:    5,
			Capacity:  10,
		},
	}
	fetcherInstance := NewPriceFloorFetcher(floorConfig, http.DefaultClient, &metricsConf.NilMetricsEngine{})
	defer fetcherInstance.Stop()

	fetchConfig := config.AccountPriceFloors{
		Enabled:        true,
		UseDynamicData: true,
		Fetcher: config.AccountFloorFetch{
			Enabled:       true,
			URL:           "http://test.com/floors",
			Timeout:       100,
			MaxFileSizeKB: 1000,
			MaxRules:      100,
			MaxAge:        5,
			Period:        1,
		},
	}

	data, status := fetcherInstance.Fetch(fetchConfig)

	assert.Equal(t, (*openrtb_ext.PriceFloorRules)(nil), data, "floor data should be nil as fetcher instance does not created")
	assert.Equal(t, openrtb_ext.FetchNone, status, "floor status should be none as fetcher instance does not created")
}
