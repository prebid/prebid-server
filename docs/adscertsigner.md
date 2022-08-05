##Ads Cert

Ads Cert is an experimental feature to support Ads.Cert 2.0 in Prebid Server.
The ads.cert protocol provides a standard method for distributing public keys so that other ads
ecosystem participants can find them and use them within these key exchange and message
authentication processes. To simplify this process, we use the domain name system (DNS) to
distribute public keys. 

Detailed Ads.Cert 2.0 specification is published on the [IAB Tech Lab ads.cert website](https://iabtechlab.com/ads-cert).


###General set up
According to [Ads Cert Authenticated Connections protocol](https://iabtechlab.com/wp-content/uploads/2021/09/3-ads-cert-authenticated-connections-pc.pdf) 
the requested domain requires to support Call Sign Internet domain established for Public keys publishing. 
In case origin URL is **bidder.com** then two subdomains has to be configured to return TXT records: 

`_adscert.bidder.com` - returns record in next format:
`v=adpf a=bidder.com`

`_delivery._adscert.bidder.com` - returns record that looks like this:
`v=adcrtd k=x25519 h=sha256 p=w8f3160kEklY-nKuxogvn5PsZQLfkWWE0gUq_4JfFm8`

For testing purposes please use this test domain (subscription will expire in May 2023):
`adscertdelivery.com`. To check data it returns use any online tool ([like this](https://mxtoolbox.com/SuperTool.aspx), select TXT lookup) to read TXT records: 
`_delivery._adscert.adscertdelivery.com` and `_adscert.adscertdelivery.com`

Or just run cli command:
```dig txt _delivery._adscert.adscertdelivery.com``` 

Public key returned in `_delivery._adscert.adscertdelivery.com` was generated using [OSS repository](https://github.com/IABTechLab/adscert).
From the project root compile sources and run `go run . basicinsecurekeygen`. This will return randomly generated private and public keys and the entire value for `_delivery._adscert.adscertdelivery.com` record.

Private key for public key published under `_delivery._adscert.adscertdelivery.com`:
```
Randomly generated key pair
Public key:  HweE1-dFJPjHO4C34QXq6myhtMuyi4X0T2rUolVzQig
Private key: U6KBGSEQ5kuMn3s_ohxYbmdmG7Xoos9hR3fJ_dDOi6Q
DNS TXT Entry: "v=adcrtd k=x25519 h=sha256 p=HweE1-dFJPjHO4C34QXq6myhtMuyi4X0T2rUolVzQig"
```

If everything configured correctly then `X-Ads-Cert-Auth` header will be sent to bidder. Detailed information about content of the header value can be found in Ads Cert Authenticated Connections protocol specification.

###Prebid Server set up
Current Prebid Server implementation supports in-process and remote signing approach.

####In-Process signer
To enable AdsCerts next configurations should be specified: 

Host config, can be set using env variables or yaml config, use proper format: 
```json
"experiment": {
    "adscert": {
      "mode": "inprocess",
      "inprocess": {
        "origin": "http://adscertdelivery.com",
        "key": "U6KBGSEQ5kuMn3s_ohxYbmdmG7Xoos9hR3fJ_dDOi6Q",
        "domain_check_interval_seconds": 30,
        "domain_renewal_interval_seconds": 30
      }
    }
  }
```
####Remote signer
To use this approach standalone GRPC server should be available.
One way to do this is to run in locally. For this checkout [AdsCert OSS](https://github.com/IABTechLab/adscert) and navigate to https://github.com/IABTechLab/adscert/blob/main/cmd/server/main.go file.
Modify L17, set "origin" to `adscertdelivery.com`, make sure ports 3000 and 3001 are available and run main function.
In Prebid Server configs set parameters for this server: 
```json
"experiment": {
    "adscert": {
      "mode": "remote",
      "remote": {
        "url": "localhost:3000",
        "signing_timeout_ms": 5
      }
    }
  }
```

####General Prebid Server set up
Workaround for bidders that don't have Call Signs support yet: in configs modify bidder URL to `http://adscertdelivery.com/openrtb2?prebid_disabled=1`. In this case this bidder will not return bids, because this endpoint doesn't exist, but it will imitate support of Call Signs. Bidder parameters still should be valid.

Every bidder by default doesn't support AdsCert. Some bidders cannot handle unsupported headers properly. To enable this feature add next config to {bidder}.yaml file:
`experiment.adsCert.enabled: true`. With this config bidder will receive `X-Ads-Cert-Auth` header even if this is not the only bidder in request. 

Request extension should have `request.ext.prebid.experiment.adscert.enabled: true`

###Issue to fix:
- After server start up the very first request doesn't have `X-Ads-Cert-Auth` header. But it works every time after the first request.
- Bidders that don't support CallSigns don't receive a default `X-Ads-Cert-Auth` header