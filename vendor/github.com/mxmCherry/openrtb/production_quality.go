package openrtb

// 5.13 Production Quality
//
// Options for content quality.
// These values are defined by the IAB; refer to www.iab.com/wp-content/uploads/2015/03/long-form-video-final.pdf for more information.
type ProductionQuality int8

const (
	ProductionQualityUnknown                ProductionQuality = 0 // Unknown
	ProductionQualityProfessionallyProduced ProductionQuality = 1 // Professionally Produced
	ProductionQualityProsumer               ProductionQuality = 2 // Prosumer
	ProductionQualityUserGenerated          ProductionQuality = 3 // User Generated (UGC)
)

// Ptr returns pointer to own value.
func (q ProductionQuality) Ptr() *ProductionQuality {
	return &q
}

// Val safely dereferences pointer, returning default value (ProductionQualityUnknown) for nil.
func (q *ProductionQuality) Val() ProductionQuality {
	if q == nil {
		return ProductionQualityUnknown
	}
	return *q
}
