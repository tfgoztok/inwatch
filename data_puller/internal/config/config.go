package config

import (
	"context"
	"data_puller/internal/model"
)

// ConfigManager defines the methods for managing configuration changes.
type ConfigManager interface {
	// Initialize sets up the configuration manager.
	Initialize(ctx context.Context) error

	// WatchChanges listens for configuration changes and returns a channel to receive them.
	WatchChanges(ctx context.Context) (<-chan ConfigChange, error)

	// GetDevices retrieves a list of devices.
	GetDevices(ctx context.Context) ([]*model.Device, error)

	// GetDeviceQueries retrieves queries associated with a specific device.
	GetDeviceQueries(ctx context.Context, deviceID int64) ([]*model.DeviceQuery, error)
}

// ConfigChange represents a change in configuration.
type ConfigChange struct {
	Type     string      // Type of change: "device_added", "device_updated", "device_removed", "query_updated"
	DeviceID int64       // ID of the device associated with the change
	Data     interface{} // Additional data related to the change
}
