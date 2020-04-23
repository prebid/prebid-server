package consentconstants

// Purpose is one of the IAB GDPR purposes. These appear in:
//   1. `root.purposes[i]` of the vendor list: https://vendorlist.consensu.org/vendorlist.json
//   2. PurposesAllowed of the Consent string: https://github.com/InteractiveAdvertisingBureau/GDPR-Transparency-and-Consent-Framework/blob/master/Consent%20string%20and%20vendor%20list%20formats%20v1.1%20Final.md#vendor-consent-string-format-
type Purpose uint8

// TCF 1 Purposes:
const (
	// InfoStorageAccess includes the storage of information, or access to information that is already stored,
	// on your device such as advertising identifiers, device identifiers, cookies, and similar technologies.
	InfoStorageAccess Purpose = 1

	// Personalization includes the collection and processing of information about your use of this service to subsequently personalise
	// advertising and/or content for you in other contexts, such as on other websites or apps, over time.
	// Typically, the content of the site or app is used to make inferences about your interests, which inform
	// future selection of advertising and/or content.
	Personalization Purpose = 2

	// AdSelectionDeliveryReporting includes the collection of information, and combination with previously collected information, to select and deliver
	// advertisements for you, and to measure the delivery and effectiveness of such advertisements. This includes using previously
	// collected information about your interests to select ads, processing data about what advertisements were shown, how often they were shown,
	// when and where they were shown, and whether you took any action related to the advertisement, including for example clicking an ad or making a purchase.
	// This does not include personalisation, which is the collection and processing of information about your use of this service to
	// subsequently personalise advertising and/or content for you in other contexts, such as websites or apps, over time.
	AdSelectionDeliveryReporting Purpose = 3

	// ContentSelectionDeliveryReporting includes the collection of information, and combination with previously collected information, to select and deliver
	// content for you, and to measure the delivery and effectiveness of such content. This includes using previously
	// collected information about your interests to select content, processing data about what content was shown,
	// how often or how long it was shown, when and where it was shown, and whether the you took any action related to
	// the content, including for example clicking on content. This does not include personalisation, which is the collection
	// and processing of information about your use of this service to subsequently personalise content and/or advertising
	// for you in other contexts, such as websites or apps, over time.
	ContentSelectionDeliveryReporting Purpose = 4

	// Measurement includes the collection of information about your use of the content, and combination with previously collected information,
	// used to measure, understand, and report on your usage of the service. This does not include personalisation, the
	// collection of information about your use of this service to subsequently personalise content and/or advertising
	// for you in other contexts, i.e. on other service, such as websites or apps, over time.
	Measurement Purpose = 5
)
