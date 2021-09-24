package richaudience

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "http://ortb.richaudience.com/ortb/?bidder=pbs",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudiencetest", bidder)
}

func TestGetBuilder(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "http://ortb.richaudience.com/ortb/?bidder=pbs"})

	if buildErr != nil {
		t.Errorf("error %s", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudience", bidder)
}

func TestGetSite(t *testing.T) {
	raBidRequest := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "www.test.com",
		},
	}

	richaudienceRequestTest := &richaudienceRequest{
		Site: richaudienceSite{
			Domain: "www.test.com",
		},
	}

	setSite(raBidRequest, richaudienceRequestTest)

	if raBidRequest.Site.Domain != richaudienceRequestTest.Site.Domain {
		t.Errorf("error %s", richaudienceRequestTest.Site.Domain)
	}

	raBidRequest = &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "http://www.test.com/test",
			Domain: "",
		},
	}

	richaudienceRequestTest = &richaudienceRequest{
		Site: richaudienceSite{
			Domain: "",
		},
	}

	setSite(raBidRequest, richaudienceRequestTest)

	if "" == richaudienceRequestTest.Site.Domain {
		t.Errorf("error domain is diferent %s", richaudienceRequestTest.Site.Domain)
	}
}

func TestGetDevice(t *testing.T) {

	raBidRequest := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP: "11.222.33.44",
			UA: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		},
	}

	richaudienceRequestTest := &richaudienceRequest{
		Device: richaudienceDevice{
			IP:  "11.222.33.44",
			Lmt: 0,
			DNT: 0,
			UA:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		},
	}

	setDevice(raBidRequest, richaudienceRequestTest)

	if raBidRequest.Device.IP != richaudienceRequestTest.Device.IP {
		t.Errorf("error %s", richaudienceRequestTest.Device.IP)
	}

	if richaudienceRequestTest.Device.Lmt == 1 {
		t.Errorf("error %v", richaudienceRequestTest.Device.Lmt)
	}

	if richaudienceRequestTest.Device.DNT == 1 {
		t.Errorf("error %v", richaudienceRequestTest.Device.DNT)
	}

	if raBidRequest.Device.UA != richaudienceRequestTest.Device.UA {
		t.Errorf("error %s", richaudienceRequestTest.Device.UA)
	}
}

func TestGetRequest(t *testing.T) {
	raBidder := new(RichaudienceAdapter)

	richaudienceRequestTest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "12345678",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 250, H: 300},
				},
			},
			Ext: json.RawMessage(`{"pid":"OsNsyeF68q","supplyType":"site","testRa":true, "bidfloor": 1}`)},
		},
		Device: &openrtb2.Device{
			H:  300,
			W:  250,
			UA: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
			IP: "111.22.33.444",
		},
		Site: &openrtb2.Site{
			ID:     "12345678",
			Domain: "bridge.richmediastudio.com",
			Page:   "http://bridge.richmediastudio.com/",
		},
		User: &openrtb2.User{
			BuyerUID: "189f4055-78a3-46eb-b7fd-0915a1a43bd2a",
			Ext: json.RawMessage(`{
				"eids": [
				  {
					"source": "id5-sync.com",
					"uids": [
					  {
						"id": "ID5-ZHMOC5mEw0TZiThiUevdyq0gjh7Egh3A4p4i9XGP-w!ID5*IJg4jnFJ8QI-Cfz5GIGeHLB9VU9kFPfcujLr44-h-joAAKkVfCHD0kZJQMomGpGV",
						"atype": 1,
						"ext": {
						  "linkType": 2,
						  "abTestingControlGroup": false
						}
					  }
					]
				  }
				],
				"consent": "CPItQrlPItQrlAKAfAENBhCsAP_AAHLAAAiQIBtf_X__bX9j-_59f_t0eY1P9_r_v-Qzjhfdt-8N2L_W_L0X42E7NF3apq4KuR4Eu3LBIQNlHMHUTUmw6okVrzPsak2Mr7NKJ7LEmnMZe2dYGHtfn91TuZKY7_78_9fz3z-v_v___9f3r-3_3__59X---_e_V399zLv9_____9nN_4HKgEmGpfABZiWOBJNGlUKIEIVhIdACACigGFomsICBwU7KwCPUEDABAagIwIgQYgoxYBAAAAAEhEQEgB4IBEARAIAAQAqQEIACJAAFgBIGAQACgGhIARQBCBIQRGBUcpgQESLRQTyRgCUXexhhCGUUAFAo_gAA.YAAAAAAAAAAA",
				"ConsentedProvidersSettings": {
				  "consented_providers": "1~39.43.46.55.61.66.70.83.89.93.108.117.122.124.131.135.136.143.144.147.149.159.162.167.171.192.196.202.211.218.228.230.239.241.253.259.266.272.286.291.311.317.322.323.326.327.338.367.371.385.389.394.397.407.413.415.424.430.436.440.445.448.449.453.482.486.491.494.495.501.503.505.522.523.540.550.559.560.568.574.576.584.587.591.733.737.745.780.787.802.803.817.820.821.829.839.853.864.867.874.899.904.922.931.938.979.981.985.1003.1024.1027.1031.1033.1034.1040.1046.1051.1053.1067.1085.1092.1095.1097.1099.1107.1127.1135.1143.1149.1152.1162.1166.1186.1188.1192.1201.1205.1211.1215.1226.1227.1230.1252.1268.1270.1276.1284.1286.1290.1301.1307.1312.1329.1345.1356.1364.1365.1375.1403.1411.1415.1416.1419.1440.1442.1449.1455.1456.1465.1495.1512.1516.1525.1540.1548.1555.1558.1564.1570.1577.1579.1583.1584.1591.1603.1616.1638.1651.1653.1665.1667.1677.1678.1682.1697.1699.1703.1712.1716.1721.1722.1725.1732.1745.1750.1765.1769.1782.1786.1800.1808.1810.1825.1827.1832.1837.1838.1840.1842.1843.1845.1859.1866.1870.1878.1880.1889.1899.1917.1929.1942.1944.1962.1963.1964.1967.1968.1969.1978.2003.2007.2008.2027.2035.2039.2044.2046.2047.2052.2056.2064.2068.2070.2072.2074.2088.2090.2103.2107.2109.2115.2124.2130.2133.2137.2140.2145.2147.2150.2156.2166.2177.2179.2183.2186.2202.2205.2216.2219.2220.2222.2225.2234.2253.2264.2279.2282.2292.2299.2305.2309.2312.2316.2325.2328.2331.2334.2335.2336.2337.2343.2354.2357.2358.2359.2366.2370.2376.2377.2387.2392.2394.2400.2403.2405.2407.2411.2414.2416.2418.2425.2427.2440.2447.2459.2461.2462.2468.2472.2477.2481.2484.2486.2488.2492.2493.2496.2497.2498.2499.2501.2510.2511.2517.2526.2527.2532.2534.2535.2542.2544.2552.2563.2564.2567.2568.2569.2571.2572.2575.2577.2583.2584.2589.2595.2596.2601.2604.2605.2608.2609.2610.2612.2614.2621.2628.2629.2633.2634.2636.2642.2643.2645.2646.2647.2650.2651.2652.2656.2657.2658.2660.2661.2669.2670.2677.2681.2684.2686.2687.2690.2695.2698.2707.2713.2714.2729.2739.2767.2768.2770.2771.2772.2784.2787.2791.2792.2798.2801.2805.2812.2813.2816.2817.2818.2821.2822.2827.2830.2831.2834.2836.2838.2839.2840.2844.2846.2847.2849.2850.2851.2852.2854.2856.2860.2862.2863.2865.2867.2869.2873.2874.2875.2876.2878.2879.2880.2881.2882.2883.2884.2885.2886.2887.2888.2889.2891.2893.2894.2895.2897.2898.2900.2901.2908.2909.2911.2912.2913.2914.2916.2917.2918.2919.2920.2922.2923.2924.2927.2929.2930.2931.2939.2940.2941.2942.2947.2949.2950.2956.2961.2962.2963.2964.2965.2966.2968.2970.2973.2974.2975.2979.2980.2981.2983.2985.2986.2987.2991.2993.2994.2995.2997.3000.3002.3003.3005.3008.3009.3010.3011.3012.3016.3017.3018.3019.3024.3025.3034.3037.3038.3043.3044.3045.3048.3052.3053.3055.3058.3059.3063.3065.3066.3068.3070.3072.3073.3074.3075.3076.3077.3078.3089.3090.3093.3094.3095.3097.3099.3100.3104.3106.3109.3111.3112.3116.3117.3118.3119.3120.3121.3124.3126.3127.3128.3130.3135.3136.3145.3149.3150.3151.3154.3155.3159.3162.3163.3167.3172.3173.3180.3182.3183.3184.3185.3187.3188.3189.3190.3194.3196.3197.3209.3210.3211.3214.3215.3217.3219.3222.3223.3225.3226.3227.3228.3230.3231.3232.3234.3235.3236.3237.3238.3240.3241.3244.3245.3250.3251.3253.3257"
				}
			  }`),
		},
	}

	request, _ := raBidder.MakeRequests(richaudienceRequestTest, &adapters.ExtraRequestInfo{})

	if request != nil {
		httpReq := request[0]
		assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

		var rpRequest openrtb2.BidRequest
		if err := json.Unmarshal(httpReq.Body, &rpRequest); err != nil {
			t.Fatalf("Failed to unmarshal HTTP request: %v", rpRequest)
		}

		var rpRequestImpExt openrtb_ext.ExtImpRichaudience
		if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpRequestImpExt); err != nil {
			t.Fatalf("Failed to unmarshal ExtImp request: %v", rpRequestImpExt)
		}

		assert.Equal(t, "OsNsyeF68q", rpRequestImpExt.Pid)
		assert.Equal(t, "site", rpRequestImpExt.SupplyType)
	}
}

func TestResponseEmpty(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(RichaudienceAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected Nil")
	assert.Empty(t, errs, "Errors: %d", len(errs))
}

func TestResponseOK(t *testing.T) {
	richaudienceRequestTest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "12345678",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 250, H: 300},
				},
			},
			Ext: json.RawMessage(`{"pid":"OsNsyeF68q","supplyType":"site","testRa":true, "bidfloor": 1}`)},
		},
		Device: &openrtb2.Device{
			H:  300,
			W:  250,
			UA: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
			IP: "111.22.33.444",
		},
		Site: &openrtb2.Site{
			ID:     "12345678",
			Domain: "bridge.richmediastudio.com",
			Page:   "http://bridge.richmediastudio.com/",
		},
		User: &openrtb2.User{
			BuyerUID: "189f4055-78a3-46eb-b7fd-0915a1a43bd2a",
			Ext: json.RawMessage(`[
				{
				  "source": "id5-sync.com",
				  "uids": [
					{
					  "id": "ID5-ZHMOPwHns8cK_i5CR4_1Jn4TM63wU4550c2_9_9yVA!ID5*c6Zpphbvw1ru5NrHa6mrHA-QN9qEHndnUD_pVt4RVr0AADvnKFJrxox11QLJscB1",
					  "atype": 1,
					  "ext": {
						"linkType": 2,
						"abTestingControlGroup": false
					  }
					}
				  ]
				},
				{
				  "source": "liveintent.com",
				  "uids": [
					{
					  "id": "YFccXozX7YIfZk4W6AIofcr2RylTCfniuLzMaQ",
					  "atype": 3
					}
				  ]
				}
			  ]
		}`),
		},
	}

	requestParsed, _ := json.Marshal(richaudienceRequestTest)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "urlEndpointTest",
		Body:    requestParsed,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{
			"bid": [
			  {
				"id": "452231263",
				"impid": "div-gpt-ad-1460505748561-0",
				"price": 99,
				"adm": "<div id=\"gseRaiOsNsyeF68q\" style=\"width:300px;height:250px\"><img src=\"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=1&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\" width=\"0\" height=\"0\" style=\"float: left;\"><a target=\"_blank\" href=\"http://richaudience.com\"><img src=\"https://cdn3.richaudience.com/demo/300x250.jpeg\" width=\"\" height=\"\"></a><script>function RAAdViewability(){var t=new OAVGeometryViewabilityCalculator,e={percentObscured:0,percentViewable:0,acceptedViewablePercentage:50,viewabiltyStatus:!1,duration:0};this.DEBUG_MODE=!1,this.checkViewability=function(t,n){var o=0,r=this,a=setInterval(function(){i(t)?o++:o=0,e.duration=100*o,o>=9&&(e.viewabiltyStatus=!0,r.DEBUG_MODE||clearInterval(a)),n(e)},100)};var i=function(t){var i=t.getBoundingClientRect();return i.width*i.height>=242500&&(e.acceptedViewablePercentage=50),!0!==o(t)&&(!0!==r(t)&&(n(t),!(e.percentViewable&&e.percentViewable<e.acceptedViewablePercentage)&&!!e.percentViewable))},n=function(i){e.percentObscured=e.percentObscured||0;var n=t.getViewabilityState(i,window);return n.error||(e.percentViewable=n.percentViewable-e.percentObscured),n},o=function(t){var e=window.getComputedStyle(t,null),i=e.getPropertyValue(\"visibility\"),n=e.getPropertyValue(\"display\");try{if(void 0!==parent.document.getElementById(window.name)&&null!=parent.document.getElementById(window.name)){var o=window.getComputedStyle(parent.document.getElementById(window.name),null),r=o.getPropertyValue(\"visibility\"),a=o.getPropertyValue(\"display\");if(\"hidden\"==i||\"none\"==n||\"hidden\"==r||\"none\"==a)return!0}else if(\"hidden\"==i||\"none\"==n)return!0}catch(t){if(\"hidden\"==i||\"none\"==n)return!0}return!1},r=function(t){var i=t.getBoundingClientRect(),n=i.left+12,o=i.right-12,r=i.top+12,h=i.bottom-12,d=Math.floor(i.left+i.width/2),l=Math.floor(i.top+i.height/2),c=[{x:n,y:r},{x:d,y:r},{x:o,y:r},{x:n,y:l},{x:d,y:l},{x:o,y:l},{x:n,y:h},{x:d,y:h},{x:o,y:h}];for(var u in c)if(c[u]&&c[u].x>=0&&c[u].y>=0&&(elem=document.elementFromPoint(c[u].x,c[u].y),null!=elem&&elem!=t&&!t.contains(elem)&&(overlappingArea=a(i,elem.getBoundingClientRect()),overlappingArea>0&&(e.percentObscured=100*a(i,elem.getBoundingClientRect()),e.percentObscured>e.acceptedViewablePercentage))))return e.percentViewable=100-e.percentObscured,!0;return!1},a=function(t,e){var i=t.width*t.height;return Math.max(0,Math.min(t.right,e.right)-Math.max(t.left,e.left))*Math.max(0,Math.min(t.bottom,e.bottom)-Math.max(t.top,e.top))/i}}function OAVGeometryViewabilityCalculator(){this.getViewabilityState=function(n,o){var r,a=t();if(a.area==1/0)return{error:\"Failed to determine viewport\"};var h={width:0,height:0,area:0},d=n.getBoundingClientRect(),l=d.width*d.height;if(a.area/l<.5)r=Math.floor(100*a.area/l);else{h=e(window.top);var c=i(n,o);c.bottom>h.height&&(c.height-=c.bottom-h.height),c.top<0&&(c.height+=c.top),c.left<0&&(c.width+=c.left),c.right>h.width&&(c.width-=c.right-h.width),r=Math.floor(c.width*c.height*100/l)}return{clientWidth:h.width,clientHeight:h.height,objTop:d.top,objBottom:d.bottom,objLeft:d.left,objRight:d.right,percentViewable:r}};var t=function(){for(var t=e(window),i=t.area,n=window;n!=window.top;)n=n.parent,viewPortSize=e(n),viewPortSize.area<i&&(i=viewPortSize.area,t=viewPortSize);return t},e=function(t){var e={width:1/0,height:1/0,area:1/0};try{return!isNaN(t.document.body.clientWidth)&&t.document.body.clientWidth>0&&(e.width=t.document.body.clientWidth),!isNaN(t.document.body.clientHeight)&&t.document.body.clientHeight>0&&(e.height=t.document.body.clientHeight),t.document.documentElement&&t.document.documentElement.clientWidth&&!isNaN(t.document.documentElement.clientWidth)&&(e.width=t.document.documentElement.clientWidth),t.document.documentElement&&t.document.documentElement.clientHeight&&!isNaN(t.document.documentElement.clientHeight)&&(e.height=t.document.documentElement.clientHeight),t.innerWidth&&!isNaN(t.innerWidth)&&(e.width=Math.min(e.width,t.innerWidth)),t.innerHeight&&!isNaN(t.innerHeight)&&(e.height=Math.min(e.height,t.innerHeight)),e.area=e.height*e.width,e}catch(t){return e.width=0,e.height=0,e.area=0,e}},i=function(t,e){var o=e,r=e.parent,a={width:0,height:0,left:0,right:0,top:0,bottom:0};if(t){var h=n(t,e);if(h.width=h.right-h.left,h.height=h.bottom-h.top,a=h,o!=r){var d=i(o.frameElement,r);d.bottom<a.bottom&&(d.bottom<a.top&&(a.top=d.bottom),a.bottom=d.bottom),d.right<a.right&&(d.right<a.left&&(a.left=d.right),a.right=d.right),a.width=a.right-a.left,a.height=a.bottom-a.top}}return a},n=function(t,e){var i=e,o=e.parent,r={left:0,right:0,top:0,bottom:0};if(t){var a=t.getBoundingClientRect();i!=o&&(r=n(i.frameElement,o)),r={left:a.left+r.left,right:a.right+r.left,top:a.top+r.top,bottom:a.bottom+r.top}}return r}}var DEBUG = false    //Attention settings\n    window.raiAttentionTimeOsNsyeF68q = 0;\n    window.raiAttentionTotalTimeOsNsyeF68q = 0;\n    window.raiAttentionIntervalOsNsyeF68q = 2;\n    window.raiAttentionExecutionOsNsyeF68q = 0;\n    window.raiAttentionPercentViewableOsNsyeF68q = 0;\n    //Attention tracking system\n    window.trackingAttentionOsNsyeF68q= function () {\n        //Check viewability percentage\n        oavAttention = new RAAdViewability();\n        oavAttention.DEBUG_MODE = false;\n        oavAttention.checkViewability(document.getElementById(\"gseRaiOsNsyeF68q\"), function(check){\n            window.raiAttentionPercentViewableOsNsyeF68q = check.percentViewable;\n        });\n        //Attention tracking mesurement\n        if(window.raiAttentionPercentViewableOsNsyeF68q > 60 && !document.hidden){\n            //timer increment\n            window.raiAttentionTimeOsNsyeF68q++;\n            if(window.raiAttentionTimeOsNsyeF68q == window.raiAttentionIntervalOsNsyeF68q){\n                //update total execution time\n                window.raiAttentionTotalTimeOsNsyeF68q = window.raiAttentionTotalTimeOsNsyeF68q + window.raiAttentionIntervalOsNsyeF68q;\n                //reset timer\n                window.raiAttentionTimeOsNsyeF68q = 0;\n                //update timer interval\n                if(window.raiAttentionExecutionOsNsyeF68q == 4){\n                    window.raiAttentionIntervalOsNsyeF68q = 5;\n                }else{\n                    if(window.raiAttentionExecutionOsNsyeF68q == 8){\n                        window.raiAttentionIntervalOsNsyeF68q = 10;\n                    }else{\n                        if(window.raiAttentionExecutionOsNsyeF68q == 11){\n                            //finish attention interval tracking\n                            clearInterval(window.IntervalAttentionOsNsyeF68q);\n                        }\n                    }\n                }\n                //attention tracking execution\n                var urlTA=\"https://t2.richaudience.com/?e=3&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&tm=\"+window.raiAttentionTotalTimeOsNsyeF68q;\n                var request = createCORSRequest(\"get\", urlTA);\n                if (request){\n                    request.onload = function(){\n                        if (DEBUG){console.log('%c SEND Attention Tracking: '+window.raiAttentionTotalTimeOsNsyeF68q,'background: lightgreen; color: red','OsNsyeF68q');}\n                    };\n                    request.send();\n                }\n                //increment execution counter\n                window.raiAttentionExecutionOsNsyeF68q++;\n            }\n        }\n    }\n    \nraimpresionOsNsyeF68q = false;\nraviewOsNsyeF68q = false;\nraIgniteLoadedImpresionOsNsyeF68q = false;\nraIgniteEngagementLoadOsNsyeF68q = false;\n\nraIsTPCOsNsyeF68q = false;\nraIsFFOsNsyeF68q = false;\n\n\ntopParentOsNsyeF68q = window;\n\ntry{\n\n    while(topParentOsNsyeF68q != topParentOsNsyeF68q.window.parent && topParentOsNsyeF68q.window.parent.document != null){\n        topParentOsNsyeF68q = topParentOsNsyeF68q.window.parent;\n    }\n\n    raIsFFOsNsyeF68q = true;\n\n}catch(e){\n    if(topParentOsNsyeF68q.location.href.indexOf('tpc.google')){\n        raIsTPCOsNsyeF68q = true;\n    }\n}\n\nvar DEBUG = false;var isIgnite = 0;\n\nvar raTargetOsNsyeF68q;\nvar topParent = [];\nvar topRef = [];\nvar raIsTPC = [];\ntopParent[\"OsNsyeF68q\"] = window;\n    try {\n        while (topParent[\"OsNsyeF68q\"] != topParent[\"OsNsyeF68q\"].window.parent && topParent[\"OsNsyeF68q\"].window.parent.document != null) {\n            topRef[\"OsNsyeF68q\"] = topParent[\"OsNsyeF68q\"].window.frameElement;\n            topParent[\"OsNsyeF68q\"] = topParent[\"OsNsyeF68q\"].window.parent;\n        }\n    } catch (e) {\n        if (topParent[\"OsNsyeF68q\"].location.href.indexOf('tpc.google')) {\n            raIsTPC[\"OsNsyeF68q\"] = true;\n        }\n    }\nif(isIgnite){\n    window.rmsIgniteContext=\"b708cc\";\n    window.raIgniteQueueb708cc = [];\n    window.raIgniteIntOsNsyeF68q = [];\n    window.rmsRAPlaHashb708cc=\"OsNsyeF68q\";\n    window.rmsRASiteHashb708cc=\"cibdzW768N\";\n}\nif(typeof window.igniteCallbackb708cc != 'function'){\n    window.igniteCallbackb708cc=function(){\n        try{\n            if(typeof raIgniteQueueb708cc != 'undefined' && raIgniteQueueb708cc.length > 0){\n                for (const fn of raIgniteQueueb708cc) {\n                    eval(fn+'()');\n                }\n            }\n        }catch(e){\n            \n        }\n    }\n}\n\nif(raIsTPCOsNsyeF68q==false){\n    \n    /*Ignite settings*/\n    raIsVisibleSO = function (el,raScrollOffset){\n        if (typeof topParent[\"OsNsyeF68q\"].document.getElementById(\"gseRaiOsNsyeF68q\") !== 'undefined') {\n            el = topParent[\"OsNsyeF68q\"].document.getElementById(\"gseRaiOsNsyeF68q\");\n        }\n        if(window == window.parent){\n            el = document.getElementById('gseRaiOsNsyeF68q');\n        }else{\n            if (typeof topRef[\"OsNsyeF68q\"] !== 'undefined') {\n                el = topRef[\"OsNsyeF68q\"];\n            }\n        }\n        var rect = el.getBoundingClientRect();\n                \n        if((rect.top >= 0) && (((rect.bottom - rect.height) - raScrollOffset) <= topParent[\"OsNsyeF68q\"].innerHeight)){\n            return true;\n        }\n        \n        if((rect.bottom >= 0) && (((rect.top + rect.height)) <= topParent[\"OsNsyeF68q\"].innerHeight)){\n            return true;\n        }\n        \n        return false;\n    }\n    window.raiIgniteSendLoadedImpressionOsNsyeF68q = true;\n    window.raiIgniteSendVisualEngagementLoadOsNsyeF68q = true;\n    window.raiIgniteVisibleOsNsyeF68q = false;\n    /*Ignite tracking system*/\n    if(window.raiIgniteSendVisualEngagementLoadOsNsyeF68q){\n        /*oavIgnite = new RAAdViewability();\n        oavIgnite.DEBUG_MODE = false;*/\n        if (typeof topParent[\"OsNsyeF68q\"].document.getElementById(\"gseRaiOsNsyeF68q\") !== 'undefined') {\n            raTargetOsNsyeF68q = topParent[\"OsNsyeF68q\"].document.getElementById(\"gseRaiOsNsyeF68q\");\n        }\n        if(window == window.parent){\n            raTargetOsNsyeF68q = document.getElementById('gseRaiOsNsyeF68q');\n        }else{\n            if (typeof topRef[\"OsNsyeF68q\"] !== 'undefined') {\n                raTargetOsNsyeF68q = topRef[\"OsNsyeF68q\"];\n            }\n        }\n        window.raIgniteIntervalOsNsyeF68q = function () {\n        /*oavIgnite.checkViewability(raTargetOsNsyeF68q, function(check){*/       \n            if (typeof googleWonOsNsyeF68q !== 'undefined') {\n                if(googleWonOsNsyeF68q==true){\n                    var igniteLoadedImpressionOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=40&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=5&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=1&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=3&cmpId=&opt_type=0\";\n                    var igniteVisualEngagementLoadOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=41&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=5&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=1&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=3&cmpId=&opt_type=0\";\n                }else{\n                    var igniteLoadedImpressionOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=40&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n                    var igniteVisualEngagementLoadOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=41&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n                }\n            }else{\n                var igniteLoadedImpressionOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=40&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n                var igniteVisualEngagementLoadOsNsyeF68q = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=41&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n            }\n            \n            if(raIsVisibleSO(raTargetOsNsyeF68q,150) && window.raiIgniteSendLoadedImpressionOsNsyeF68q){\n                if(isIgnite){\n                    if (typeof fireLoadedImpresionb708cc == 'function' ){\n                        fireLoadedImpresionb708cc();\n                    }else{\n                        raIgniteQueueb708cc.push('fireLoadedImpresionb708cc');\n                    }\n                    \n                    if (typeof fireLoadedImpresionRA == 'function' ){\n                        fireLoadedImpresionRA();\n                    }else{\n                        raIgniteIntOsNsyeF68q.push('fireLoadedImpresionRA');\n                    }\n                }\n                window.raiIgniteSendLoadedImpressionOsNsyeF68q = false;\n\n                if(igniteLoadedImpressionOsNsyeF68q!=false && igniteLoadedImpressionOsNsyeF68q!=\"\") {\n    \n                    var request = createCORSRequest(\"get\", igniteLoadedImpressionOsNsyeF68q);\n                    if (request){\n                        request.onload = function(){\n                            if (DEBUG){console.log('%c SEND IgniteLoadedImpression Tracking:'+igniteLoadedImpressionOsNsyeF68q,'background: lightgreen; color: red','OsNsyeF68q');}\n                        };\n                        request.send();\n                    }\n                }\n            }\n            if(raIsVisibleSO(raTargetOsNsyeF68q,\"-\"+raTargetOsNsyeF68q.getBoundingClientRect().height/5) > 0 && window.raiIgniteSendVisualEngagementLoadOsNsyeF68q){\n            /*if(check.percentViewable > 20 && window.raiIgniteSendVisualEngagementLoadOsNsyeF68q){*/\n                if(isIgnite){\n                    if (typeof fireVisualEngLoadb708cc == 'function' ){\n                        fireVisualEngLoadb708cc();\n                    }else{\n                        raIgniteQueueb708cc.push('fireVisualEngLoadb708cc');\n                    }\n                }\n                window.raiIgniteSendVisualEngagementLoadOsNsyeF68q = false;\n                var request = createCORSRequest(\"get\", igniteVisualEngagementLoadOsNsyeF68q);\n                if (request){\n                    request.onload = function(){\n                        if (DEBUG){console.log('%c SEND IgniteVisualEngagementLoad Tracking:'+igniteVisualEngagementLoadOsNsyeF68q,'background: lightgreen; color: red','OsNsyeF68q');}\n                    };\n                    request.send();\n                }\n                /*clearInterval(window.raIgniteIntervalOsNsyeF68q);*/\n                topParent[\"OsNsyeF68q\"].removeEventListener('scroll', window.raIgniteIntervalOsNsyeF68q);\n            }\n        /*});*/\n        };\n    }\n    topParent[\"OsNsyeF68q\"].addEventListener('scroll', window.raIgniteIntervalOsNsyeF68q);\n}\nif(raIsFFOsNsyeF68q == true){\n    urlImp = \"https://t2.richaudience.com/?e=1&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=1\";\n    urlView = \"https://t2.richaudience.com/?e=2&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=1\";\n}else{\n    if(typeof topParentOsNsyeF68q.$sf != \"undefined\" && typeof topParentOsNsyeF68q.$sf.ext != \"undefined\" && typeof topParentOsNsyeF68q.$sf.ext.supports != \"undefined\"  &&  topParentOsNsyeF68q.$sf.ext.supports() != null  && topParentOsNsyeF68q.$sf.ext.supports()['exp-push'] == true){\n        urlImp = \"https://t2.richaudience.com/?e=1&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=2\";\n        urlView = \"https://t2.richaudience.com/?e=2&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=2\";\n    }else{\n        urlImp = \"https://t2.richaudience.com/?e=1&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=3\";\n        urlView = \"https://t2.richaudience.com/?e=2&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3&intgr=3\";\n    }\n}\n\n\n\nif(raIsFFOsNsyeF68q == true){\n\n\n\nfunction createCORSRequest(method, url){\n    var xhr = new XMLHttpRequest();\n    if (\"withCredentials\" in xhr){\n        xhr.open(method, url, true);\n    } else if (typeof XDomainRequest != \"undefined\"){\n        xhr = new XDomainRequest();\n        xhr.open(method, url);\n    } else {\n        xhr = null;\n    }\n    return xhr;\n}\n\noav = new RAAdViewability();\noav.DEBUG_MODE = false;\n\n\n\n}\n\n\n\nif(raIsTPCOsNsyeF68q == true){\n\n    if(typeof topParentOsNsyeF68q.$sf != \"undefined\" && typeof topParentOsNsyeF68q.$sf.ext != \"undefined\" && typeof topParentOsNsyeF68q.$sf.ext.inViewPercentage != \"undefined\"){\n\n            if(raimpresionOsNsyeF68q == false){\n                raImgReqOsNsyeF68q = document.createElement('img');\n                raImgReqOsNsyeF68q.style.width = \"0\";\n                raImgReqOsNsyeF68q.style.height = \"0\";\n                raImgReqOsNsyeF68q.src = urlImp;\n                document.body.appendChild(raImgReqOsNsyeF68q);\n                raimpresionOsNsyeF68q = true;\n            }\n\n            raViewSecOsNsyeF68q = 0;\n\n\n\n            //Attention settings\n            var raiAttentionTimeOsNsyeF68q = 0;\n            var raiAttentionTotalTimeOsNsyeF68q = 0;\n            var raiAttentionIntervalOsNsyeF68q = 2;\n            var raiAttentionExecutionOsNsyeF68q = 0;\n\n            var raiSendAttTrackingOsNsyeF68q = Math.floor((Math.random() * 100))<10;\n\n            raViewIntOsNsyeF68q = setInterval(function(){\n                                //Ignite Loaded Impression\n                if(topParentOsNsyeF68q.$sf.ext.inViewPercentage() >= 1 && !document.hidden && raIgniteLoadedImpresionOsNsyeF68q==false){\n                    raImgIgniteImpOsNsyeF68q = document.createElement('img');\n                    raImgIgniteImpOsNsyeF68q.style.width = \"0\";\n                    raImgIgniteImpOsNsyeF68q.style.height = \"0\";\n                    raImgIgniteImpOsNsyeF68q.src = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=40&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n                    document.body.appendChild(raImgIgniteImpOsNsyeF68q);\n                    raIgniteLoadedImpresionOsNsyeF68q = true;\n                                    }\n                                //Ignite Visual Engagement Load\n                if(topParentOsNsyeF68q.$sf.ext.inViewPercentage() >= 20 && !document.hidden && raIgniteEngagementLoadOsNsyeF68q==false){\n                    raImgIgniteImpOsNsyeF68q = document.createElement('img');\n                    raImgIgniteImpOsNsyeF68q.style.width = \"0\";\n                    raImgIgniteImpOsNsyeF68q.style.height = \"0\";\n                    raImgIgniteImpOsNsyeF68q.src = \"https://t.richaudience.com?s=18892&p=OsNsyeF68q&cn=0&e=41&lct=1e8919c275cbae1ba7bff62b2b993cf9&jpt=1e8919c275cbae1ba7bff62b2b993cf9&stn=1e8919c275cbae1ba7bff62b2b993cf9&mrc=1e8919c275cbae1ba7bff62b2b993cf9&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&type=3&subtype=1&idplatform=0&env_id=2&pid=127315&gdpr_con=1&advd=&nde=&raplayer=0&prebid=2&tc=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&dsp=0&rmshash=0&dt=0&cmpId=&opt_type=0\";\n                    document.body.appendChild(raImgIgniteImpOsNsyeF68q);\n                                        raIgniteEngagementLoadOsNsyeF68q = true;\n                }\n\n                if(topParentOsNsyeF68q.$sf.ext.inViewPercentage() >= 50){\n                    raViewSecOsNsyeF68q++;\n                }else{\n                    raViewSecOsNsyeF68q=0;\n                }\n\n                if(raViewSecOsNsyeF68q >= 1){\n                    if(raimpresionOsNsyeF68q == true && raviewOsNsyeF68q == false){\n                        raImgImpOsNsyeF68q = document.createElement('img');\n                        raImgImpOsNsyeF68q.style.width = \"0\";\n                        raImgImpOsNsyeF68q.style.height = \"0\";\n                        raImgImpOsNsyeF68q.src = urlView;\n                        document.body.appendChild(raImgImpOsNsyeF68q);\n                        //clearInterval(raViewIntOsNsyeF68q);\n                        raviewOsNsyeF68q = true;\n                                            }\n                    //Attention tracking system\n                    if(raiSendAttTrackingOsNsyeF68q){\n                        if(raViewSecOsNsyeF68q >= 2){\n                            if(topParentOsNsyeF68q.$sf.ext.inViewPercentage() >= 100 && !document.hidden){\n                                //timer increment\n                                raiAttentionTimeOsNsyeF68q++;\n                                if(raiAttentionTimeOsNsyeF68q == raiAttentionIntervalOsNsyeF68q){\n                                    //update total execution time\n                                    raiAttentionTotalTimeOsNsyeF68q = raiAttentionTotalTimeOsNsyeF68q + raiAttentionIntervalOsNsyeF68q;\n                                    //reset timer\n                                    raiAttentionTimeOsNsyeF68q = 0;\n                                    //update timer interval\n                                    if(raiAttentionExecutionOsNsyeF68q == 4){\n                                        raiAttentionIntervalOsNsyeF68q = 5;\n                                    }else{\n                                        if(raiAttentionExecutionOsNsyeF68q == 8){\n                                            raiAttentionIntervalOsNsyeF68q = 10;\n                                        }else{\n                                            if(raiAttentionExecutionOsNsyeF68q == 11){\n                                                //finish attention interval tracking\n                                                clearInterval(raViewIntOsNsyeF68q);\n                                            }\n                                        }\n                                    }\n                                    //attention tracking execution\n                                    raImgImpOsNsyeF68q = document.createElement('img');\n                                    raImgImpOsNsyeF68q.style.width = \"0\";\n                                    raImgImpOsNsyeF68q.style.height = \"0\";\n                                    raImgImpOsNsyeF68q.src = \"https://t2.richaudience.com/?e=3&p=OsNsyeF68q&s=18892&type=3&subtype=1&wscs=&hscs=&ua=Mozilla%2F5.0+%28Macintosh%3B+Intel+Mac+OS+X+10_15_7%29+AppleWebKit%2F537.36+%28KHTML%2C+like+Gecko%29+Chrome%2F91.0.4472.114+Safari%2F537.36&tscs=&inw=&inh=&wou=&hou=&sgn=fJP6wjqlkf%2FwnDJDY76KuWTskz2i8hh4yhZw%2BhRHUI0XwSJWGyR%2F67kHMfDHACWqIpk4MMmCNlVkg9Sv6ogVmoCG34RwhWJBPuhe9OYGPvb6dSHIRdwZWySNu5McxhOV0ibR1uSPQQSQlohVSyakAWbGeTPLVBp1Y0qlia9dXWCW9CLDwrN%2BYCuUphlEau62OsYktIde%2BMPJNGTn6XYyMh0aZHpCwXLxkxZeRjY1R8L4GgHjsknHawQJzn65OV2ymCnyDhTiq45%2FYGVRwCE0%2BdoSQ4fyvZGGjLKETcIel3l%2F%2FDUPwlSsesk%2F&v=15badaf9aeea2682e69fcad4f392b33cc66fd2e6c12ac148b5333be5b020d27a&dt=3\"+\"&tm=\"+raiAttentionTotalTimeOsNsyeF68q;\n                                    document.body.appendChild(raImgImpOsNsyeF68q);\n                                    //increment execution counter\n                                    raiAttentionExecutionOsNsyeF68q++;\n                                }\n                            }\n                        }\n                    }else{\n                        //finish attention interval tracking\n                        clearInterval(raViewIntOsNsyeF68q);\n                    }\n                }\n            }, 1000);\n    }\n}\n</script></div>",
				"adomain": [
				  "richaudience.com"
				],
				"crid": "999999",
				"w": 300,
				"h": 250,
				"ext": {
				  "prebid": {
					"targeting": {
					  "hb_bidder": "richaudience",
					  "hb_pb": "20.00",
					  "hb_size": "300x250"
					},
					"type": "banner"
				  }
				}
			  }
			],
			"seat": "richaudience"
		  }`),
	}

	bidder := new(RichaudienceAdapter)
	bidResponse, errs := bidder.MakeBids(richaudienceRequestTest, reqData, httpResp)

	t.Log(bidResponse)
	t.Log(errs)
}

func TestBadConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint:         `https://test.ortb`,
		ExtraAdapterInfo: `{foo:42}`,
	})

	assert.Empty(t, bidder)
	assert.NoError(t, buildErr)
}

func TestEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint:         ``,
		ExtraAdapterInfo: ``,
	})

	assert.NoError(t, buildErr)
	assert.Empty(t, bidder)
}
