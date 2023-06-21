/*
	builder-struct-name tests
	https://semgrep.dev/docs/writing-rules/testing-rules
	"ruleid" prefix in comment indicates patterns that should be flagged by semgrep
	"ok" prefix in comment indidcates  patterns that should not be flagged by the semgrep
*/

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	foo1 := foo{}
	// ruleid: builder-struct-name-check
	return &fooadapter{foo: foo1}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &adapterbar{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &fooadapterbar{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &FooAdapter{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &AdapterBar{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &AdapterBar{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	// ruleid: builder-struct-name-check
	return &FooAdapterBar{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	foo2 := foo{}
	//ruleid: builder-struct-name-check
	adpt1 := Adapter{foo: foo2}
	return &adpt1, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	//ruleid: builder-struct-name-check
	builder := &Adapter{foo{}}
	return builder, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	foo3 := foo{}
	if foo3.bar == "" {
		foo3.bar = "bar"
	}
	//ruleid: builder-struct-name-check
	adpt2 := Adapter{}
	adpt2.foo = foo3
	return &adpt2, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	//ruleid: builder-struct-name-check
	return &foo{}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	//ruleid: builder-struct-name-check
	var obj Adapter
	obj.Foo = "foo"
	if obj.Bar == "" {
		obj.Bar = "bar"
	}
	return &obj, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	//ruleid: builder-struct-name-check
	var obj *FooAdapterBar
	obj.Foo = "foo"
	if obj.Bar == "" {
		obj.Bar = "bar"
	}
	return obj, nil
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	// ok: builder-struct-name-check
	return &adapter{endpoint: "www.foo.com"}, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	builder := &adapter{}
	builder.endpoint = "www.foo.com"
	// ok: builder-struct-name-check
	return builder, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	builder := adapter{}
	builder.endpoint = "www.foo.com"
	// ok: builder-struct-name-check
	return &builder, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	var builder adapter
	builder.endpoint = "www.foo.com"
	// ok: builder-struct-name-check
	return &builder, nil
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	var builder *adapter
	builder.endpoint = "www.foo.com"
	// ok: builder-struct-name-check
	return builder, nil
}