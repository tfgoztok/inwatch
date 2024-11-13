package model

// Device represents a network device with its attributes.
type Device struct {
	ID       int64  `json:"id"`       // Unique identifier for the device
	Name     string `json:"name"`     // Name of the device
	IP       string `json:"ip"`       // IP address of the device
	Port     int    `json:"port"`     // Port number for the device
	Protocol string `json:"protocol"` // Communication protocol used by the device
	Status   string `json:"status"`   // Current status of the device
}

// DeviceQuery represents a query related to a specific device.
type DeviceQuery struct {
	ID            int64  `json:"id"`             // Unique identifier for the query
	DeviceID      int64  `json:"device_id"`      // Identifier of the associated device
	QueryString   string `json:"query_string"`   // The actual query string
	ParameterName string `json:"parameter_name"` // Name of the parameter being queried
	DataType      string `json:"data_type"`      // Type of data expected from the query
	PollInterval  int    `json:"poll_interval"`  // Interval for polling the device
	Enabled       bool   `json:"enabled"`        // Indicates if the query is enabled
}

// DeviceConfig represents the configuration settings for a device.
type DeviceConfig struct {
	ID          int64  `json:"id"`           // Unique identifier for the configuration
	DeviceID    int64  `json:"device_id"`    // Identifier of the associated device
	ConfigKey   string `json:"config_key"`   // Key for the configuration setting
	ConfigValue string `json:"config_value"` // Value for the configuration setting
}
