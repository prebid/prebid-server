package openrtb_ext

import (
    "github.com/mxmCherry/openrtb"
    "encoding/json"
)

// This will copy the openrtb BidRequest into an array of requests, where the BidRequest.Imp[].Ext field will only
// consist of the "prebid" field and the field for the appropriate bidder parameters. We will drop all extended fields
// beyond this context, so this will not be compatible with any other uses of the extension area. That is, this routine
// will work, but the adapters will not see any other extension fields.

// Take an openrtb request, and a list of bidders, and return an openrtb request sanitized for each bidder
func CleanOpenRTBRequests(orig *openrtb.BidRequest, adapters []string) []*openrtb.BidRequest {
    // This is the clean array of openrtb requests we will be returning
    clean_reqs := make([]*openrtb.BidRequest, len(adapters))

    // Decode the Imp extensions once to save time. We store the results here
    imp_exts := make([]map[string]interface{}, len(orig.Imp) )
    // Loop over every impression in the request
    for i := 0 ; i < len(orig.Imp) ; i++ {
        // Unpack each set of extensions found in the Imp array
        err := json.Unmarshal(orig.Imp[i].Ext, &imp_exts[i])
        _ = err
        // Need to do some error handling here here
    }

    // Loop over every adapter we want to create a clean openrtb request for.
    for i := 0 ; i < len(adapters); i++ {
        // Create a new BidRequest
        new_req := new(openrtb.BidRequest)
        // Make a shallow copy of the original request
        *new_req = *orig
        // Go deeper into Imp array
        new_Imp := make([]openrtb.Imp, len(orig.Imp))
        copy(new_Imp, orig.Imp)

        // Overwrite each extension field with a cleanly built subset
        // We are looping over every impression in the Imp array
        for j := 0 ; j < len(orig.Imp) ; j++ {
            // Start with a new, empty unpacked extention
            new_ext := map[string]interface{}{}
            // Need to do some consistency checking to verify these fields exist. Especially the adapters one.
            if val, ok := imp_exts[j]["prebid"]; ok {
                new_ext["prebid"] = val
            }
            if val, ok := imp_exts[j][adapters[i]]; ok {
                new_ext[adapters[i]] = val
            }
            // Create a "clean" byte array for this Imp's extension
            // Note, if the "prebid" or "<adapter>" field is missing from the source, it will be missing here as well
            // The adapters should test that their field is present rather than assuming it will be there if they are
            // called
            b, err := json.Marshal(new_ext)
            // Need error handling here
            _ = err
            // Overwrite the extention field with the new cleaned version
            new_Imp[j].Ext = b
        }
        new_req.Imp = new_Imp
        clean_reqs[i] = new_req
    }
    return clean_reqs
}