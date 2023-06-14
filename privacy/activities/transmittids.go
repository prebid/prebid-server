package activities

import (
	"encoding/json"
	"github.com/prebid/openrtb/v19/openrtb2"
)

func RemoveTIds(bidReq *openrtb2.BidRequest) error {

	//!!!only for testing
	bidReq.ID = "modified!!!"

	if bidReq.Source != nil {
		bidReq.Source.TID = ""
	}

	for i, imp := range bidReq.Imp {
		if len(imp.Ext) > 0 {
			var err error
			//!!!better way to search and remove?
			var impExt map[string]interface{}
			err = json.Unmarshal(imp.Ext, &impExt)
			if err != nil {
				return err
			}

			if _, present := impExt["tid"]; present {
				delete(impExt, "tid")
				newExt, err := json.Marshal(impExt)
				if err != nil {
					return err
				}
				bidReq.Imp[i].Ext = newExt
			}

		}
	}
	return nil
}
