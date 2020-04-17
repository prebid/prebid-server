package openrtb

// 5.15 Content Delivery Methods
//
// Various options for the delivery of video or audio content.
type ContentDeliveryMethod int8

const (
	ContentDeliveryMethodStreaming   ContentDeliveryMethod = 1 // Streaming
	ContentDeliveryMethodProgressive ContentDeliveryMethod = 2 // Progressive
	ContentDeliveryMethodDownload    ContentDeliveryMethod = 3 // Download
)
