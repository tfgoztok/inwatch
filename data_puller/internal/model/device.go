package model

// Device represents a network device with its attributes.
type Device struct {
	ID       int64  `json:"id"`       // Unique identifier for the device
	Name     string `json:"name"`     // Name of the device
	IP       string `json:"ip"`       // IP address of the device
	Port     int    `json:"port"`     // Port number used by the device
	Protocol string `json:"protocol"` // Communication protocol used (e.g., TCP, UDP)
	Status   string `json:"status"`   // Current status of the device (e.g., online, offline)
}

// DeviceQuery represents a query related to a specific device.
type DeviceQuery struct {
	ID            int64  `json:"id"`             // Unique identifier for the query
	DeviceID      int64  `json:"device_id"`      // Identifier of the associated device
	QueryString   string `json:"query_string"`   // Query string (e.g., OID)
	ParameterName string `json:"parameter_name"` // Name of the parameter being queried
	DataType      string `json:"data_type"`      // Type of data expected (e.g., "string", "int", "float")
}
