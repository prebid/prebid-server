package tmp

import (
	"encoding/hex"
	"encoding/json"
	"hash"
	"sync"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// contextCacheKey derives a stable hex string from inputs that scope a Context
// Match result. Same (property_rid, placement_id, page/app, privacy-safe ids)
// returns the same key.
func contextCacheKey(pool *sync.Pool, propertyRID, placementID string, br *openrtb2.BidRequest) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("p:" + propertyRID))
	_, _ = h.Write([]byte("|pl:" + placementID))
	writeSiteOrApp(h, br)
	writePrivacySafeUserIDs(h, br.User)
	return hex.EncodeToString(h.Sum(nil))
}

// identityCacheKey derives a stable hex string from inputs that scope an
// Identity Match result. Identity match results are page-context-free, so the
// key intentionally excludes site/app/placement.
func identityCacheKey(pool *sync.Pool, sellerAgentURL, country string, idents []IdentityToken) string {
	h := pool.Get().(hash.Hash)
	defer pool.Put(h)
	h.Reset()

	_, _ = h.Write([]byte("s:" + sellerAgentURL))
	_, _ = h.Write([]byte("|c:" + country))
	for _, t := range idents {
		_, _ = h.Write([]byte("|id:" + t.UIDType + "=" + t.UserToken))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeSiteOrApp(h hash.Hash, br *openrtb2.BidRequest) {
	if br.Site != nil {
		_, _ = h.Write([]byte("|d:" + br.Site.Domain))
		if br.Site.Page != "" {
			_, _ = h.Write([]byte("|pg:" + br.Site.Page))
		}
	}
	if br.App != nil {
		_, _ = h.Write([]byte("|a:" + br.App.Bundle))
	}
}

func writePrivacySafeUserIDs(h hash.Hash, user *openrtb2.User) {
	if user == nil {
		return
	}
	var ext struct {
		EIDs []openrtb2.EID `json:"eids"`
	}
	if len(user.Ext) > 0 {
		_ = json.Unmarshal(user.Ext, &ext)
	}
	for _, eid := range ext.EIDs {
		if len(eid.UIDs) > 0 {
			_, _ = h.Write([]byte("|e:" + eid.Source + "=" + eid.UIDs[0].ID))
		}
	}
}

// intersect returns the package IDs that appear in both the context offers
// list and the identity-eligible list. Order follows the contextOffers; output
// is deduplicated. Returns an empty (non-nil) slice when either input is empty.
func intersect(contextOffers []Offer, identityEligible []string) []string {
	out := []string{}
	if len(contextOffers) == 0 || len(identityEligible) == 0 {
		return out
	}
	eligible := make(map[string]struct{}, len(identityEligible))
	for _, id := range identityEligible {
		eligible[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(contextOffers))
	for _, offer := range contextOffers {
		if _, alreadyEmitted := seen[offer.PackageID]; alreadyEmitted {
			continue
		}
		if _, ok := eligible[offer.PackageID]; ok {
			out = append(out, offer.PackageID)
			seen[offer.PackageID] = struct{}{}
		}
	}
	return out
}
