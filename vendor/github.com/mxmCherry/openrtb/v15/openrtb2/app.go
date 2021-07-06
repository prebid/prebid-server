package openrtb2

import "encoding/json"

// 3.2.14 Object: App
//
// This object should be included if the ad supported content is a non-browser application (typically in mobile) as opposed to a website.
// A bid request must not contain both an App and a Site object.
// At a minimum, it is useful to provide an App ID or bundle, but this is not strictly required.
type App struct {

	// Attribute:
	//   id
	// Type:
	//   string; recommended
	// Description:
	//   Exchange-specific app ID.
	ID string `json:"id,omitempty"`

	// Attribute:
	//   name
	// Type:
	//   string
	// Description:
	//   App name (may be aliased at the publisher’s request).
	Name string `json:"name,omitempty"`

	// Attribute:
	//   bundle
	// Type:
	//   string
	// Description:
	//   A platform-specific application identifier intended to be
	//   unique to the app and independent of the exchange. On
	//   Android, this should be a bundle or package name (e.g.,
	//   com.foo.mygame). On iOS, it is typically a numeric ID.
	Bundle string `json:"bundle,omitempty"`

	// Attribute:
	//   domain
	// Type:
	//   string
	// Description:
	//   Domain of the app (e.g., “mygame.foo.com”).
	Domain string `json:"domain,omitempty"`

	// Attribute:
	//   storeurl
	// Type:
	//   string
	// Description:
	//   App store URL for an installed app; for IQG 2.1 compliance.
	StoreURL string `json:"storeurl,omitempty"`

	// Attribute:
	//   cat
	// Type:
	//   string array
	// Description:
	//   Array of IAB content categories of the app. Refer to List 5.1
	Cat []string `json:"cat,omitempty"`

	// Attribute:
	//   sectioncat
	// Type:
	//   string array
	// Description:
	//   Array of IAB content categories that describe the current
	//   section of the app. Refer to List 5.1.
	SectionCat []string `json:"sectioncat,omitempty"`

	// Attribute:
	//   pagecat
	// Type:
	//   string array
	// Description:
	//   Array of IAB content categories that describe the current page
	//   or view of the app. Refer to List 5.1.
	PageCat []string `json:"pagecat,omitempty"`

	// Attribute:
	//   ver
	// Type:
	//   string
	// Description:
	//   Application version.
	Ver string `json:"ver,omitempty"`

	// Attribute:
	//   privacypolicy
	// Type:
	//   integer
	// Description:
	//   Indicates if the app has a privacy policy, where 0 = no, 1 = yes.
	PrivacyPolicy int8 `json:"privacypolicy,omitempty"`

	// Attribute:
	//   paid
	// Type:
	//   integer
	// Description:
	//   0 = app is free, 1 = the app is a paid version.
	Paid int8 `json:"paid,omitempty"`

	// Attribute:
	//   publisher
	// Type:
	//   object
	// Description:
	//   Details about the Publisher (Section 3.2.15) of the app.
	Publisher *Publisher `json:"publisher,omitempty"`

	// Attribute:
	//   content
	// Type:
	//   object
	// Description:
	//   Details about the Content (Section 3.2.16) within the app
	Content *Content `json:"content,omitempty"`

	// Attribute:
	//   keywords
	// Type:
	//   string
	// Description:
	//   Comma separated list of keywords about the app.
	Keywords string `json:"keywords,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
