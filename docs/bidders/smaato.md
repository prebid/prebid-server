
# Smaato Bidder

```
Module Name: Smaato Bidder Adapter
Module Type: Bidder Adapter
Maintainer: prebid@smaato.com
```

### Description

Please contact Smaato Support or prebid@smaato.com to get set up with a publisherId and adspaceId.

### Test Parameters:  

Following example includes sample `imp` object with publisherId and adSlot which can be used to test Smaato Adapter

```
"imp":[
      {
         "id":“1C86242D-9535-47D6-9576-7B1FE87F282C,
         "banner":{
            "format":[
               {
                  "w":300,
                  "h":50
               },
               {
                  "w":300,
                  "h":250
               }
            ]
         },
         "ext":{
            "smaato":{
               "publisherId":"100042525",
               "adspaceId":"130563103"
            }
         }
      }
   ]
```
