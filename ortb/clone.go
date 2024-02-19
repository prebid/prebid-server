package ortb

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
	"github.com/prebid/prebid-server/v2/util/sliceutil"
)

func CloneApp(s *openrtb2.App) *openrtb2.App {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = sliceutil.Clone(s.Cat)
	c.SectionCat = sliceutil.Clone(s.SectionCat)
	c.PageCat = sliceutil.Clone(s.PageCat)
	c.PrivacyPolicy = ptrutil.Clone(s.PrivacyPolicy)
	c.Paid = ptrutil.Clone(s.Paid)
	c.Publisher = ClonePublisher(s.Publisher)
	c.Content = CloneContent(s.Content)
	c.KwArray = sliceutil.Clone(s.KwArray)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func ClonePublisher(s *openrtb2.Publisher) *openrtb2.Publisher {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = sliceutil.Clone(s.Cat)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneContent(s *openrtb2.Content) *openrtb2.Content {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Producer = CloneProducer(s.Producer)
	c.Cat = sliceutil.Clone(s.Cat)
	c.ProdQ = ptrutil.Clone(s.ProdQ)
	c.VideoQuality = ptrutil.Clone(s.VideoQuality)
	c.KwArray = sliceutil.Clone(s.KwArray)
	c.LiveStream = ptrutil.Clone(s.LiveStream)
	c.SourceRelationship = ptrutil.Clone(s.SourceRelationship)
	c.Embeddable = ptrutil.Clone(s.Embeddable)
	c.Data = CloneDataSlice(s.Data)
	c.Network = CloneNetwork(s.Network)
	c.Channel = CloneChannel(s.Channel)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneProducer(s *openrtb2.Producer) *openrtb2.Producer {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = sliceutil.Clone(s.Cat)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneDataSlice(s []openrtb2.Data) []openrtb2.Data {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.Data, len(s))
	for i, d := range s {
		c[i] = CloneData(d)
	}

	return c
}

func CloneData(s openrtb2.Data) openrtb2.Data {
	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	// - Implicitly created by the cloned array.

	// Deep Copy (Pointers)
	s.Segment = CloneSegmentSlice(s.Segment)
	s.Ext = sliceutil.Clone(s.Ext)

	return s
}

func CloneSegmentSlice(s []openrtb2.Segment) []openrtb2.Segment {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.Segment, len(s))
	for i, d := range s {
		c[i] = CloneSegment(d)
	}

	return c
}

func CloneSegment(s openrtb2.Segment) openrtb2.Segment {
	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	// - Implicitly created by the cloned array.

	// Deep Copy (Pointers)
	s.Ext = sliceutil.Clone(s.Ext)

	return s
}

func CloneNetwork(s *openrtb2.Network) *openrtb2.Network {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneChannel(s *openrtb2.Channel) *openrtb2.Channel {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneSite(s *openrtb2.Site) *openrtb2.Site {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = sliceutil.Clone(s.Cat)
	c.SectionCat = sliceutil.Clone(s.SectionCat)
	c.PageCat = sliceutil.Clone(s.PageCat)
	c.Mobile = ptrutil.Clone(s.Mobile)
	c.PrivacyPolicy = ptrutil.Clone(s.PrivacyPolicy)
	c.Publisher = ClonePublisher(s.Publisher)
	c.Content = CloneContent(s.Content)
	c.KwArray = sliceutil.Clone(s.KwArray)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneUser(s *openrtb2.User) *openrtb2.User {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.KwArray = sliceutil.Clone(s.KwArray)
	c.Geo = CloneGeo(s.Geo)
	c.Data = CloneDataSlice(s.Data)
	c.EIDs = CloneEIDSlice(s.EIDs)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneDevice(s *openrtb2.Device) *openrtb2.Device {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Geo = CloneGeo(s.Geo)
	c.DNT = ptrutil.Clone(s.DNT)
	c.Lmt = ptrutil.Clone(s.Lmt)
	c.SUA = CloneUserAgent(s.SUA)
	c.JS = ptrutil.Clone(s.JS)
	c.GeoFetch = ptrutil.Clone(s.GeoFetch)
	c.ConnectionType = ptrutil.Clone(s.ConnectionType)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneUserAgent(s *openrtb2.UserAgent) *openrtb2.UserAgent {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Browsers = CloneBrandVersionSlice(s.Browsers)
	c.Platform = CloneBrandVersion(s.Platform)

	if s.Mobile != nil {
		mobileCopy := *s.Mobile
		c.Mobile = &mobileCopy
	}
	s.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneBrandVersionSlice(s []openrtb2.BrandVersion) []openrtb2.BrandVersion {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.BrandVersion, len(s))
	for i, d := range s {
		bv := CloneBrandVersion(&d)
		c[i] = *bv
	}

	return c
}

func CloneBrandVersion(s *openrtb2.BrandVersion) *openrtb2.BrandVersion {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	c := *s

	// Deep Copy (Pointers)
	c.Version = sliceutil.Clone(s.Version)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneSource(s *openrtb2.Source) *openrtb2.Source {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.FD = ptrutil.Clone(s.FD)
	c.SChain = CloneSChain(s.SChain)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneSChain(s *openrtb2.SupplyChain) *openrtb2.SupplyChain {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Nodes = CloneSupplyChainNodes(s.Nodes)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneSupplyChainNodes(s []openrtb2.SupplyChainNode) []openrtb2.SupplyChainNode {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.SupplyChainNode, len(s))
	for i, d := range s {
		c[i] = CloneSupplyChainNode(d)
	}

	return c
}

func CloneSupplyChainNode(s openrtb2.SupplyChainNode) openrtb2.SupplyChainNode {
	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	// - Implicitly created by the cloned array.

	// Deep Copy (Pointers)
	s.HP = ptrutil.Clone(s.HP)
	s.Ext = sliceutil.Clone(s.Ext)

	return s
}

func CloneGeo(s *openrtb2.Geo) *openrtb2.Geo {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Lat = ptrutil.Clone(s.Lat)
	c.Lon = ptrutil.Clone(s.Lon)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneEIDSlice(s []openrtb2.EID) []openrtb2.EID {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.EID, len(s))
	for i, d := range s {
		c[i] = CloneEID(d)
	}

	return c
}

func CloneEID(s openrtb2.EID) openrtb2.EID {
	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	// - Implicitly created by the cloned array.

	// Deep Copy (Pointers)
	s.UIDs = CloneUIDSlice(s.UIDs)
	s.Ext = sliceutil.Clone(s.Ext)

	return s
}

func CloneUIDSlice(s []openrtb2.UID) []openrtb2.UID {
	if s == nil {
		return nil
	}

	c := make([]openrtb2.UID, len(s))
	for i, d := range s {
		c[i] = CloneUID(d)
	}

	return c
}

func CloneUID(s openrtb2.UID) openrtb2.UID {
	// Shallow Copy (Value Fields) Occurred By Passing Argument By Value
	// - Implicitly created by the cloned array.

	// Deep Copy (Pointers)
	s.Ext = sliceutil.Clone(s.Ext)

	return s
}

func CloneDOOH(s *openrtb2.DOOH) *openrtb2.DOOH {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.VenueType = sliceutil.Clone(s.VenueType)
	c.VenueTypeTax = ptrutil.Clone(s.VenueTypeTax)
	c.Publisher = ClonePublisher(s.Publisher)
	c.Content = CloneContent(s.Content)
	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

// CloneBidRequestPartial performs a deep clone of just the bid request device, user, and source fields.
func CloneBidRequestPartial(s *openrtb2.BidRequest) *openrtb2.BidRequest {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers) - PARTIAL CLONE
	c.Device = CloneDevice(s.Device)
	c.User = CloneUser(s.User)
	c.Source = CloneSource(s.Source)

	return &c
}
