package ortb

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/util/ptrutil"
	"github.com/prebid/prebid-server/util/sliceutil"
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
	// Shallow Copy (Value Fields)
	// - Already occurred implicitly in the method call.

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
	// Shallow Copy (Value Fields)
	// - Already occurred implicitly in the method call.

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
