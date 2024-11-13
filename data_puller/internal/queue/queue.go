package queue

import (
	"context"
	"time"

	"data_puller/internal/model"
)

// Work represents a unit of work to be processed, containing a device and a query.
type Work struct {
	Device *model.Device      // Pointer to the Device associated with the work
	Query  *model.DeviceQuery // Pointer to the DeviceQuery associated with the work
}

// WorkQueue defines the interface for a queue that handles Work items.
type WorkQueue interface {
	Push(ctx context.Context, work *Work) error                             // Adds a Work item to the queue
	Pop(ctx context.Context) (*Work, error)                                 // Removes and returns the next Work item from the queue
	Schedule(ctx context.Context, work *Work, interval time.Duration) error // Schedules a Work item to be processed after a specified interval
	Cancel(deviceID int64) error                                            // Cancels any scheduled Work for the given device ID
}
