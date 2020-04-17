package openrtb

// 5.22 Connection Type
//
// Various options for the type of device connectivity.
type ConnectionType int8

const (
	ConnectionTypeUnknown                          ConnectionType = 0 // Unknown
	ConnectionTypeEthernet                         ConnectionType = 1 // Ethernet
	ConnectionTypeWIFI                             ConnectionType = 2 // WIFI
	ConnectionTypeCellularNetworkUnknownGeneration ConnectionType = 3 // Cellular Network – Unknown Generation
	ConnectionTypeCellularNetwork2G                ConnectionType = 4 // Cellular Network – 2G
	ConnectionTypeCellularNetwork3G                ConnectionType = 5 // Cellular Network – 3G
	ConnectionTypeCellularNetwork4G                ConnectionType = 6 // Cellular Network – 4G
)

// Ptr returns pointer to own value.
func (t ConnectionType) Ptr() *ConnectionType {
	return &t
}

// Val safely dereferences pointer, returning default value (ConnectionTypeUnknown) for nil.
func (t *ConnectionType) Val() ConnectionType {
	if t == nil {
		return ConnectionTypeUnknown
	}
	return *t
}
