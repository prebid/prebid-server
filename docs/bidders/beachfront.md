# Beachfront bidder

To use the beachfront bidder you will need an appId (Exchange Id) from an exchange 
account on [platform.beachfront.io](https://platform.beachfront.io).

For further information, please contact adops@beachfront.com.

As seen in the JSON response from \{your PBS server\}\/bidder\/params [(example)](https://prebid.adnxs.com/pbs/v1/bidders/params), the beachfront bidder can take either an "appId" parameter, or an "appIds" parameter. If the request is for one media type, the appId parameter should be used with the value of the Exchange Id on the Beachfront platform.

The appIds parameter is for requesting a mix of banner and video. It has two parameters, "banner", and "video" for the appIds of two appropriately configured exchanges on the platform. The appIds parameter can be sent with just one of its two parameters and it will behave like the appId parameter.

If the request includes an appId configured for a video response, the videoResponseType parameter can be defined as "nurl", "adm" or "both". These will apply to all video returned. If it is not defined, the response type will be a nurl. The definitions for "nurl" vs. "adm" are here: (https://github.com/mxmCherry/openrtb/blob/master/openrtb2/bid.go).

