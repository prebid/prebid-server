package ortb

import (
	"slices"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

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
	s.Ext = slices.Clone(s.Ext)

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
	s.Ext = slices.Clone(s.Ext)

	return s
}

func CloneUser(s *openrtb2.User) *openrtb2.User {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.KwArray = slices.Clone(s.KwArray)
	c.Geo = CloneGeo(s.Geo)
	c.Data = CloneDataSlice(s.Data)
	c.EIDs = CloneEIDSlice(s.EIDs)
	c.Ext = slices.Clone(s.Ext)

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
	c.Ext = slices.Clone(s.Ext)

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
	s.Ext = slices.Clone(s.Ext)

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
	c.Version = slices.Clone(s.Version)
	c.Ext = slices.Clone(s.Ext)

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
	c.Ext = slices.Clone(s.Ext)

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
	c.Ext = slices.Clone(s.Ext)

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
	s.Ext = slices.Clone(s.Ext)

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
	c.Ext = slices.Clone(s.Ext)

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
	s.Ext = slices.Clone(s.Ext)

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
	s.Ext = slices.Clone(s.Ext)

	return s
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

func CloneRegs(s *openrtb2.Regs) *openrtb2.Regs {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.GDPR = ptrutil.Clone(s.GDPR)
	c.GPPSID = slices.Clone(s.GPPSID)
	c.Ext = slices.Clone(s.Ext)

	return &c
}
