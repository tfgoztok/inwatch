// internal/collector/collector.go
package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"data_puller/internal/model"
	"data_puller/internal/protocol"
	"data_puller/internal/queue"
)

// Collector manages devices and their data collection.
type Collector struct {
	mu          sync.RWMutex
	devices     map[int64]*model.Device
	handlers    map[string]protocol.Handler
	queue       *queue.WorkQueue
	rateLimiter *queue.RateLimiter
	retryPolicy queue.RetryPolicy
	configs     map[int64]map[string]string // deviceID -> config map
}

// NewCollector creates a new Collector instance with specified queue size and worker count.
func NewCollector(queueSize, workerCount int) *Collector {
	return &Collector{
		devices:     make(map[int64]*model.Device),
		handlers:    make(map[string]protocol.Handler),
		queue:       queue.NewWorkQueue(workerCount, queueSize),
		rateLimiter: queue.NewRateLimiter(time.Second),
		retryPolicy: queue.DefaultRetryPolicy(),
		configs:     make(map[int64]map[string]string),
	}
}

// RegisterHandler registers a new protocol handler for the collector.
func (c *Collector) RegisterHandler(protocol string, handler protocol.Handler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.handlers[protocol]; exists {
		return fmt.Errorf("handler for protocol %s already registered", protocol)
	}

	c.handlers[protocol] = handler
	return nil
}

// Start initializes handlers and starts the work queue.
func (c *Collector) Start(ctx context.Context) error {
	// Initialize handlers
	for _, handler := range c.handlers {
		if err := handler.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize handler: %w", err)
		}
	}

	// Start work queue
	if err := c.queue.Start(ctx); err != nil {
		return fmt.Errorf("failed to start work queue: %w", err)
	}

	// Setup work processor
	go c.processWork(ctx)

	return nil
}

// processWork continuously processes work items from the queue.
func (c *Collector) processWork(ctx context.Context) {
	for work := range c.queue.WorkChan() {
		go c.handleWork(ctx, work)
	}
}

// handleWork processes a single work item, applying rate limiting and executing the handler.
func (c *Collector) handleWork(ctx context.Context, work *queue.Work) {
	// Rate limiting
	if err := c.rateLimiter.Wait(ctx, work.Device.ID); err != nil {
		log.Printf("Rate limiter error for device %d: %v", work.Device.ID, err)
		return
	}

	// Get handler for protocol
	c.mu.RLock()
	handler, exists := c.handlers[work.Device.Protocol]
	//deviceConfig := c.configs[work.Device.ID]
	c.mu.RUnlock()

	if !exists {
		log.Printf("No handler found for protocol %s", work.Device.Protocol)
		return
	}

	// Execute work with retry policy
	err := c.retryPolicy.DoWithRetry(ctx, func() error {
		result, err := handler.Poll(ctx, work.Device, work.Query)
		if err != nil {
			return fmt.Errorf("poll failed: %w", err)
		}

		// Process result
		c.processResult(result)
		return nil
	})

	if err != nil {
		log.Printf("Failed to collect data from device %d (%s): %v",
			work.Device.ID, work.Device.Name, err)

		// Update device status if needed
		c.updateDeviceStatus(work.Device.ID, "error")
	}
}

// processResult handles the result of a query, logging any errors.
func (c *Collector) processResult(result *protocol.QueryResult) {
	if result.Error != nil {
		log.Printf("Error in query result for device %d, parameter %s: %v",
			result.DeviceID, result.ParameterName, result.Error)
		return
	}

	// Currently just logging the result
	log.Printf("Data collected - Device: %d, Parameter: %s, Value: %v, Timestamp: %d",
		result.DeviceID, result.ParameterName, result.Value, result.Timestamp)
}

// AddDevice adds a new device to the collector with its configurations.
func (c *Collector) AddDevice(device *model.Device, configs map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate device protocol
	if _, exists := c.handlers[device.Protocol]; !exists {
		return fmt.Errorf("unsupported protocol: %s", device.Protocol)
	}

	c.devices[device.ID] = device
	c.configs[device.ID] = configs

	return nil
}

// RemoveDevice removes a device from the collector and cancels its scheduled work.
func (c *Collector) RemoveDevice(deviceID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.devices, deviceID)
	delete(c.configs, deviceID)

	// Cancel all scheduled work for this device
	return c.queue.CancelDevice(deviceID)
}

// UpdateDevice updates an existing device's information and configurations.
func (c *Collector) UpdateDevice(device *model.Device, configs map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel existing schedules
	c.queue.CancelDevice(device.ID)

	// Update device and configs
	c.devices[device.ID] = device
	c.configs[device.ID] = configs

	return nil
}

// updateDeviceStatus updates the status of a device.
func (c *Collector) updateDeviceStatus(deviceID int64, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if device, exists := c.devices[deviceID]; exists {
		device.Status = status
	}
}

// ScheduleQuery schedules a query for a specific device.
func (c *Collector) ScheduleQuery(ctx context.Context, deviceID int64, query *model.DeviceQuery) error {
	c.mu.RLock()
	device, exists := c.devices[deviceID]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device not found: %d", deviceID)
	}

	work := &queue.Work{
		Device: device,
		Query:  query,
	}

	interval := time.Duration(query.PollInterval) * time.Second
	return c.queue.Schedule(ctx, work, interval)
}

// Stop stops all handlers and cleans up resources.
func (c *Collector) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop all handlers
	for _, handler := range c.handlers {
		if err := handler.Close(); err != nil {
			log.Printf("Error closing handler: %v", err)
		}
	}

	return nil
}
