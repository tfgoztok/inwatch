package protocol

import (
	"context"
	"data_puller/internal/model"
)

// QueryResult represents the result of a query to a device.
type QueryResult struct {
	DeviceID      int64       // Unique identifier for the device
	ParameterName string      // Name of the parameter being queried
	Value         interface{} // Value returned from the query
	Timestamp     int64       // Time when the query was executed
	Error         error       // Error encountered during the query, if any
}

// Handler defines the interface for handling device queries.
type Handler interface {
	Initialize(ctx context.Context) error                                                           // Initializes the handler with the given context
	Poll(ctx context.Context, device *model.Device, query *model.DeviceQuery) (*QueryResult, error) // Polls the device for the query result
	Validate(device *model.Device, query *model.DeviceQuery) error                                  // Validates the device and query parameters
	Close() error                                                                                   // Closes the handler and releases any resources
}
