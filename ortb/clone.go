package ortb

import (
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/util/ptrutil"
)

func CloneApp(s *openrtb2.App) *openrtb2.App {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = cloneSlice(s.Cat)
	c.SectionCat = cloneSlice(s.SectionCat)
	c.PageCat = cloneSlice(s.PageCat)
	c.Publisher = ClonePublisher(s.Publisher)
	c.Content = CloneContent(s.Content)
	c.KwArray = cloneSlice(s.KwArray)
	c.Ext = cloneSlice(s.Ext)

	return &c
}

func ClonePublisher(s *openrtb2.Publisher) *openrtb2.Publisher {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = cloneSlice(s.Cat)
	c.Ext = cloneSlice(s.Ext)

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
	c.Cat = cloneSlice(s.Cat)
	c.ProdQ = ptrutil.ClonePtr(s.ProdQ)
	c.VideoQuality = ptrutil.ClonePtr(s.VideoQuality)
	c.KwArray = cloneSlice(s.KwArray)
	c.Data = CloneDataSlice(s.Data)
	c.Network = CloneNetwork(s.Network)
	c.Channel = CloneChannel(s.Channel)
	c.Ext = cloneSlice(s.Ext)

	return &c
}

func CloneProducer(s *openrtb2.Producer) *openrtb2.Producer {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Cat = cloneSlice(s.Cat)
	c.Ext = cloneSlice(s.Ext)

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
	s.Ext = cloneSlice(s.Ext)

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
	s.Ext = cloneSlice(s.Ext)

	return s
}

func CloneNetwork(s *openrtb2.Network) *openrtb2.Network {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Ext = cloneSlice(s.Ext)

	return &c
}

func CloneChannel(s *openrtb2.Channel) *openrtb2.Channel {
	if s == nil {
		return nil
	}

	// Shallow Copy (Value Fields)
	c := *s

	// Deep Copy (Pointers)
	c.Ext = cloneSlice(s.Ext)

	return &c
}

func cloneSlice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	c := make([]T, len(s))
	copy(c, s)

	return c
}
