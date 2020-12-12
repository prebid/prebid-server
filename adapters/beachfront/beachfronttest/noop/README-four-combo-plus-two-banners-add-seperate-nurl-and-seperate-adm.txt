The test file four-combo-plus-two-banners-add-seperate-nurl-and-seperate-adm.json causes
an error similar to the error that I was seeing in the multiVideo failed PR if the httpcalls[]
element for SeperateADM is placed at index 8, matching the index of the mockrequests[] element.
Likewise for indexies 6 and 7, which are both nurl. Placing the httpcalls[] element at index 5
stops the error behavior, independent of the location of the mockrequests[] element. So this places
all adm httprequest[] elements before the nurl elements.

Experiment 1:
I will instead place all nurl httprequest[] elements before all adm httprequest[] elements and see
what happens, leaving the mockrequest[] elements as they are. Note that two of the mock request
elements create both an adm and a nurl request.

Result 1:
Totally jacked.

Experiment 2:
I will modify four-combo.json, which makes 6 http calls with 4 requests, 4 adm and 2 nurl. Currently
all the nurls are last. I will put one before the last adm, so 0 adm, 1 adm, 2 adm, 3 nurl, 4 adm, 5 nurl.

Result 2:
Similarly jacked. Details:

--- FAIL: TestJsonSamples (0.03s)
    test_json.go:285: beachfronttest/supplemental/four-combo.json: httpRequest[4].uri "https://reachms.bfmio.com/bid.json?exchange_id=videoAppId1" does not match expected "https://reachms.bfmio.com/bid.json?exchange_id=videoAppId1&prebidserver."
    test_json.go:315: beachfronttest/supplemental/four-combo.json: httpRequest[4] json did not match expected.

         {
           "cur": [
             : "USD"
           ],
           "id": "four-combo",
           "imp": [
             : {
               "bidfloor": 1.01,
               "id": "ComboBothVideoWithBannerImp2",
               "secure": 0,
               "video": {
                 "h": 150,
                 "mimes": [
                   : "video/mp4"
                 ],
                 "w": 100
               }
             }
           ],
           "site": {
             "domain": "example.com",
             "page": "http://example.com/whatever/something.html"
           }
        +  "isPrebid": true
         }
    test_json.go:285: beachfronttest/supplemental/four-combo.json: httpRequest[5].uri "https://reachms.bfmio.com/bid.json?exchange_id=videoAppId1&prebidserver" does not match expected "https://reachms.bfmio.com/bid.json?exchange_id=videoAppId1."
    test_json.go:315: beachfronttest/supplemental/four-combo.json: httpRequest[5] json did not match expected.

         {
           "cur": [
             : "USD"
           ],
           "id": "four-combo",
           "imp": [
             0: {
               "bidfloor": 1.01,
        -      "id": "ComboBothVideoWithBannerImp1",
        +      "id": "ComboBothVideoWithBannerImp2",
               "secure": 0,
               "video": {
                 "h": 150,
                 "mimes": [
                   : "video/mp4"
                 ],
                 "w": 100
               }
             }
           ],
        -  "isPrebid": true,
           "site": {
             "domain": "example.com",
             "page": "http://example.com/whatever/something.html"
           }
         }
    test_json.go:315: beachfronttest/supplemental/four-combo.json: httpRequest[6] json did not match expected.

         {
           "cur": [
             : "USD"
           ],
           "id": "four-combo",
           "imp": [
             0: {
               "bidfloor": 1.01,
        -      "id": "ComboBothVideoWithBannerImp2",
        +      "id": "ComboBothVideoWithBannerImp1",
               "secure": 0,
               "video": {
                 "h": 150,
                 "mimes": [
                   : "video/mp4"
                 ],
                 "w": 100
               }
             }
           ],
           "isPrebid": true,
           "site": {
             "domain": "example.com",
             "page": "http://example.com/whatever/something.html"
           }
         }
FAIL
