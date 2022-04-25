package consentconstants

// SpecialFeature is one of the IAB GDPR special features. These appear in:
//   1. `root.specialFeatures[i]` of the vendor list: https://vendorlist.consensu.org/vendorlist.json
//   2. SpecialFeatureOptIns of the Consent string: https://github.com/InteractiveAdvertisingBureau/GDPR-Transparency-and-Consent-Framework/blob/master/Consent%20string%20and%20vendor%20list%20formats%20v1.1%20Final.md#vendor-consent-string-format-
type SpecialFeature uint8