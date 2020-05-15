package consumable

func retrieveAd(decision decision) string {

	if decision.Contents != nil && len(decision.Contents) > 0 {
		return decision.Contents[0].Body
	}

	return ""
}
