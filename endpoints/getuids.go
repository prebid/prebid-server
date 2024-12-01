package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/usersync"

	"encoding/json"
)

type userSyncs struct {
	BuyerUIDs map[string]string `json:"buyeruids,omitempty"`
}

// NewGetUIDsEndpoint implements the /getuid endpoint which
// returns all the existing syncs for the user
func NewGetUIDsEndpoint(cfg config.HostCookie) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		cookie := usersync.ReadCookie(r, usersync.Base64Decoder{}, &cfg)
		usersync.SyncHostCookie(r, cookie, &cfg)

		userSyncs := new(userSyncs)
		userSyncs.BuyerUIDs = cookie.GetUIDs()
		json.NewEncoder(w).Encode(userSyncs)
	})
}
