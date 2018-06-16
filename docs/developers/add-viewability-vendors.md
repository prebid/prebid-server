# Adding viewability support

This document describes how to handle viewability in Prebid Server

1. Choose vendor constants: These constants should be unique. The list of existing vendor constants can be found [here](../../openrtb_ext/viewability_vendors.go)

2. Add the constants to bidder-info: The list of vendors supported by your exchange are to be added to `../../static/bidder-info/{bidder}.yaml` file 

3. Map constants to vendor urls in [this file](../../openrtb_ext/viewability_vendors.go). 

4. The adapter should be able to read the vendor constants from `bidrequest.imp[i].metric[j].vendor` and map it to the respective vendor url before making a request to the exchange.


