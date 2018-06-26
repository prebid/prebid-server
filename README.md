# Pubmatic-Adaptor 
Pubmatic adaptor in Prebid Server  


## Sample Request for legacy /auction endpoint 

pbBidder := pbs.PBSBidder{
        BidderCode: "bannerCode",
        AdUnits: []pbs.PBSAdUnit{
            {
                Code:       "unitCode",
                BidID:      "bidid",
                MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
                Sizes: []openrtb.Format{
                    {
                        W: 336,
                        H: 280,
                    },
                },
                Params: json.RawMessage(`{"publisherId": "640", 
                            "adSlot": "slot1@336x280", 
                            "kadpageurl": "www.test.com", 
                            "gender": "M",
                            "lat":40.1,
                            "lon":50.2,
                            "yob":1982,
                            "kadfloor":0.5,
                            "keywords":{
                                    "pmZoneId": "Zone1,Zone2"
                                    },  
                            "wrapper":
                                    {"version":2,
                                    "profile":595}
                                    }`),
            },
        },
    }
        
## Sample Request for /openrtb2/auction endpoint 

    request := &openrtb.BidRequest{
        ID: "12345",
        Imp: []openrtb.Imp{{
            ID: "234",
            Banner: &openrtb.Banner{
                Format: []openrtb.Format{{
                    W: 300,
                    H: 250,
                }},
            },
            Ext: openrtb.RawJSON(`{"bidder": {
                                "adSlot": "AdTag_Div1@300x250",
                                "publisherId": "1234",
                                "keywords":{
                                            "pmZoneID": "Zone1,Zone2",
                                            "preference": "sports,movies"
                                            },
                                "wrapper":{"version":1,"profile":5123}
                            }}`),
        }},
        Device: &openrtb.Device{
            UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36",
        },
        User: &openrtb.User{
            ID: "testID",
        },
        Site: &openrtb.Site{
            ID: "siteID",
        },
    }