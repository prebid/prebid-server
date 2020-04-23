package openrtb

// 5.8 Protocols
//
// Options for the various bid response protocols that could be supported by an exchange.
type Protocol int8

const (
	ProtocolVAST10         Protocol = 1  // VAST 1.0
	ProtocolVAST20         Protocol = 2  // VAST 2.0
	ProtocolVAST30         Protocol = 3  // VAST 3.0
	ProtocolVAST10Wrapper  Protocol = 4  // VAST 1.0 Wrapper
	ProtocolVAST20Wrapper  Protocol = 5  // VAST 2.0 Wrapper
	ProtocolVAST30Wrapper  Protocol = 6  // VAST 3.0 Wrapper
	ProtocolVAST40         Protocol = 7  // VAST 4.0
	ProtocolVAST40Wrapper  Protocol = 8  // VAST 4.0 Wrapper
	ProtocolDAAST10        Protocol = 9  // DAAST 1.0
	ProtocolDAAST10Wrapper Protocol = 10 // DAAST 1.0 Wrapper
)
