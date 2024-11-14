package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"data_puller/internal/model"

	"github.com/lib/pq"
)

// dbConfigManager implements dbConfigManager interface.
type dbConfigManager struct {
	db       *sql.DB           // Database connection
	listener *pq.Listener      // PostgreSQL listener for notifications
	changes  chan ConfigChange // Channel to send configuration changes
	mu       sync.RWMutex      // Mutex for concurrent access
}

// NewConfigManager initializes a new dbConfigManager with a database connection.
func NewConfigManager(dsn string) (*dbConfigManager, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create a new PostgreSQL listener for notifications
	listener := pq.NewListener(dsn,
		10*time.Second, // Minimum reconnection interval
		1*time.Minute,  // Maximum reconnection interval
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Printf("Listen error: %v\n", err)
			}
		})

	return &dbConfigManager{
		db:       db,
		listener: listener,
		changes:  make(chan ConfigChange, 100), // Buffered channel for changes
	}, nil
}

// Initialize starts listening for configuration changes.
func (m *dbConfigManager) Initialize(ctx context.Context) error {
	if err := m.listener.Listen("config_changes"); err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	// Start processing notifications in background
	go m.processNotifications(ctx)

	return nil
}

// dbNotification represents the structure of a database notification.
type dbNotification struct {
	Table    string         `json:"table"`     // Table name where the change occurred
	Action   string         `json:"action"`    // Action type (INSERT, UPDATE, DELETE)
	DeviceID int64          `json:"device_id"` // ID of the device affected
	Data     map[string]any `json:"data"`      // Data associated with the change
}

// processNotifications listens for and processes database notifications.
func (m *dbConfigManager) processNotifications(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return // Exit if context is done

		case n := <-m.listener.Notify:
			if n == nil {
				continue // Skip if notification is nil
			}

			var notification dbNotification
			if err := json.Unmarshal([]byte(n.Extra), &notification); err != nil {
				log.Printf("Failed to unmarshal notification: %v", err)
				continue
			}

			change := ConfigChange{
				DeviceID: notification.DeviceID, // Set the device ID for the change
			}

			// Determine the type of change based on the notification
			switch notification.Table {
			case "devices":
				switch notification.Action {
				case "INSERT":
					change.Type = "device_added"
				case "UPDATE":
					change.Type = "device_updated"
				case "DELETE":
					change.Type = "device_removed"
				}

				if notification.Action != "DELETE" {
					device := &model.Device{}
					if err := mapToStruct(notification.Data, device); err != nil {
						log.Printf("Failed to convert device data: %v", err)
						continue
					}
					change.Data = device // Set the device data for the change
				}

			case "device_queries":
				change.Type = "query_updated"
				if notification.Action != "DELETE" {
					query := &model.DeviceQuery{}
					if err := mapToStruct(notification.Data, query); err != nil {
						log.Printf("Failed to convert query data: %v", err)
						continue
					}
					change.Data = query // Set the query data for the change
				}

			case "device_configs":
				change.Type = "config_updated"
				if notification.Action != "DELETE" {
					config := &model.DeviceConfig{}
					if err := mapToStruct(notification.Data, config); err != nil {
						log.Printf("Failed to convert config data: %v", err)
						continue
					}
					change.Data = config // Set the config data for the change
				}
			}

			// Send to changes channel, non-blocking
			select {
			case m.changes <- change:
			default:
				log.Printf("Changes channel full, dropped notification for device %d", notification.DeviceID)
			}
		}
	}
}

// GetDevices retrieves all active devices from the database.
func (m *dbConfigManager) GetDevices(ctx context.Context) ([]*model.Device, error) {
	query := `
        SELECT id, name, ip, port, protocol, status
        FROM devices
        WHERE status = 'active'
    `

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		device := &model.Device{}
		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.IP,
			&device.Port,
			&device.Protocol,
			&device.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, device) // Append the device to the list
	}

	return devices, nil // Return the list of devices
}

// GetDeviceQueries retrieves all enabled queries for a specific device.
func (m *dbConfigManager) GetDeviceQueries(ctx context.Context, deviceID int64) ([]*model.DeviceQuery, error) {
	query := `
        SELECT id, device_id, query_string, parameter_name, data_type, poll_interval, enabled
        FROM device_queries
        WHERE device_id = $1 AND enabled = true
    `

	rows, err := m.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query device queries: %w", err)
	}
	defer rows.Close()

	var queries []*model.DeviceQuery
	for rows.Next() {
		q := &model.DeviceQuery{}
		err := rows.Scan(
			&q.ID,
			&q.DeviceID,
			&q.QueryString,
			&q.ParameterName,
			&q.DataType,
			&q.PollInterval,
			&q.Enabled,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query: %w", err)
		}
		queries = append(queries, q) // Append the query to the list
	}

	return queries, nil // Return the list of queries
}

// GetDeviceConfig retrieves the configuration for a specific device.
func (m *dbConfigManager) GetDeviceConfig(ctx context.Context, deviceID int64) (map[string]string, error) {
	query := `
        SELECT config_key, config_value
        FROM device_configs
        WHERE device_id = $1
    `

	rows, err := m.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query device configs: %w", err)
	}
	defer rows.Close()

	configs := make(map[string]string) // Map to hold configuration key-value pairs
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		configs[key] = value // Store the key-value pair in the map
	}

	return configs, nil // Return the configuration map
}

// WatchChanges returns a channel to receive configuration change notifications.
func (m *dbConfigManager) WatchChanges(ctx context.Context) (<-chan ConfigChange, error) {
	return m.changes, nil // Return the changes channel
}

// Close cleans up resources and closes the database connection.
func (m *dbConfigManager) Close() error {
	if err := m.listener.Close(); err != nil {
		log.Printf("Error closing listener: %v", err)
	}

	close(m.changes)    // Close the changes channel
	return m.db.Close() // Close the database connection
}

// Helper function to convert map to struct
func mapToStruct(data map[string]any, dest interface{}) error {
	jsonData, err := json.Marshal(data) // Marshal the map to JSON
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, dest) // Unmarshal JSON to the destination struct
}
