package collector

import (
	"context"
	"sync"
)

// Collector interface defines the methods for starting, stopping, and managing devices.
type Collector interface {
	Start(ctx context.Context) error         // Starts the collector with the provided context.
	Stop() error                             // Stops the collector.
	AddDevice(device *model.Device) error    // Adds a new device to the collector.
	RemoveDevice(deviceID int64) error       // Removes a device by its ID.
	UpdateDevice(device *model.Device) error // Updates an existing device.
}

// DataCollector struct implements the Collector interface and manages devices.
type DataCollector struct {
	mu       sync.RWMutex                // Mutex for concurrent access to devices.
	devices  map[int64]*model.Device     // Map of devices indexed by their ID.
	handlers map[string]protocol.Handler // Map of protocol handlers.
	queue    queue.WorkQueue             // Work queue for processing tasks.
}

// NewDataCollector initializes a new DataCollector with a given work queue.
func NewDataCollector(queue queue.WorkQueue) *DataCollector {
	return &DataCollector{
		devices:  make(map[int64]*model.Device),     // Initialize devices map.
		handlers: make(map[string]protocol.Handler), // Initialize handlers map.
		queue:    queue,                             // Set the work queue.
	}
}
