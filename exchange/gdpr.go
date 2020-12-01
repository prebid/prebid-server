package exchange

// ExtractGDPR will pull the gdpr flag from an openrtb request
func extractGDPR(extGDPR *int8, usersyncIfAmbiguous bool) (gdpr int) {
	if extGDPR == nil {
		if usersyncIfAmbiguous {
			gdpr = 0
		} else {
			gdpr = 1
		}
	} else {
		gdpr = int(*extGDPR)
	}
	return
}

// ExtractConsent will pull the consent string from an openrtb request
func extractConsent(extInfo AuctionExtInfo) (consent string) {
	if extInfo.UserExt != nil {
		return extInfo.UserExt.Consent
	}
	return ""

}
