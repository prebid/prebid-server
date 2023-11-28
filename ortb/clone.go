package ortb

import (
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
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

	c.DNT = CloneInt8Pointer(s.DNT)
	c.Lmt = CloneInt8Pointer(s.Lmt)

	c.SUA = CloneUserAgent(s.SUA)
	if s.ConnectionType != nil {
		connectionTypeCopy := s.ConnectionType.Val()
		c.ConnectionType = &connectionTypeCopy
	}

	c.Ext = sliceutil.Clone(s.Ext)

	return &c
}

func CloneInt8Pointer(s *int8) *int8 {
	if s == nil {
		return nil
	}
	var dntCopy int8
	dntCopy = *s
	return &dntCopy
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

	// Deep Copy (Pointers)
	s.HP = CloneInt8Pointer(s.HP)
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

// cloneBidderReq - clones bidder request and replaces req.User and req.Device and req.Source with new copies
func CloneBidderReq(req *openrtb2.BidRequest) *openrtb_ext.RequestWrapper {
	if req == nil {
		return nil
	}

	// bidder request may be modified differently per bidder based on privacy configs
	// new request should be created for each bidder request
	// pointer fields like User and Device should be cloned and set back to the request copy
	newReq := ptrutil.Clone(req)

	if req.User != nil {
		userCopy := CloneUser(req.User)
		newReq.User = userCopy
	}

	if req.Device != nil {
		deviceCopy := CloneDevice(req.Device)
		newReq.Device = deviceCopy
	}

	if req.Source != nil {
		sourceCopy := CloneSource(req.Source)
		newReq.Source = sourceCopy
	}

	reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: newReq}
	return reqWrapper
}
