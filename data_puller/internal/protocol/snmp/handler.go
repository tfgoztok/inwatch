package snmp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"data_puller/internal/model"
	"data_puller/internal/protocol"

	"github.com/gosnmp/gosnmp"
)

// SNMPHandler manages SNMP clients and their operations
type SNMPHandler struct {
	clients map[int64]*gosnmp.GoSNMP // Map of SNMP clients indexed by device ID
	mu      sync.RWMutex             // Mutex for concurrent access to clients
}

// NewSNMPHandler creates a new instance of SNMPHandler
func NewSNMPHandler() *SNMPHandler {
	return &SNMPHandler{
		clients: make(map[int64]*gosnmp.GoSNMP), // Initialize the clients map
	}
}

// Initialize prepares the SNMPHandler for use, if needed
func (h *SNMPHandler) Initialize(ctx context.Context) error {
	// Initialize handler-wide resources if needed
	return nil
}

// getClient retrieves or creates an SNMP client for the specified device
func (h *SNMPHandler) getClient(device *model.Device) (*gosnmp.GoSNMP, error) {
	h.mu.RLock() // Acquire read lock
	client, exists := h.clients[device.ID]
	h.mu.RUnlock() // Release read lock

	if exists {
		return client, nil // Return existing client if found
	}

	h.mu.Lock()         // Acquire write lock
	defer h.mu.Unlock() // Ensure write lock is released

	// Double check after acquiring write lock
	if client, exists := h.clients[device.ID]; exists {
		return client, nil // Return existing client if found
	}

	// Create a new SNMP client
	client = &gosnmp.GoSNMP{
		Target:    device.IP,
		Port:      uint16(device.Port),
		Community: "public", // This value should be retrieved from device_configs table
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(2) * time.Second,
		Retries:   3,
	}

	// Attempt to connect to the SNMP device
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to SNMP device: %w", err)
	}

	h.clients[device.ID] = client // Store the new client
	return client, nil
}

// Poll retrieves data from the SNMP device based on the query
func (h *SNMPHandler) Poll(ctx context.Context, device *model.Device, query *model.DeviceQuery) (*protocol.QueryResult, error) {
	client, err := h.getClient(device) // Get the SNMP client
	if err != nil {
		return nil, err
	}

	// Prepare the result structure
	result := &protocol.QueryResult{
		DeviceID:      device.ID,
		ParameterName: query.ParameterName,
		Timestamp:     time.Now().Unix(),
	}

	oids := []string{query.QueryString} // OID to query
	packets, err := client.Get(oids)    // Perform the SNMP GET operation
	if err != nil {
		result.Error = err
		return result, err
	}

	if len(packets.Variables) == 0 {
		result.Error = fmt.Errorf("no data received")
		return result, result.Error
	}

	// Convert the received value based on DataType
	variable := packets.Variables[0]
	value, err := convertSNMPValue(variable, query.DataType) // Convert SNMP value
	if err != nil {
		result.Error = err
		return result, err
	}

	result.Value = value // Store the converted value
	return result, nil
}

// Validate checks if the device and query are valid for SNMP
func (h *SNMPHandler) Validate(device *model.Device, query *model.DeviceQuery) error {
	if device.Protocol != "snmp" {
		return fmt.Errorf("invalid protocol: expected snmp, got %s", device.Protocol)
	}

	// Check OID format
	if !isValidOID(query.QueryString) {
		return fmt.Errorf("invalid OID format: %s", query.QueryString)
	}

	return nil
}

// Close releases resources held by the SNMPHandler
func (h *SNMPHandler) Close() error {
	h.mu.Lock()         // Acquire write lock
	defer h.mu.Unlock() // Ensure write lock is released

	for _, client := range h.clients {
		client.Conn.Close() // Close each SNMP client connection
	}
	h.clients = make(map[int64]*gosnmp.GoSNMP) // Clear the clients map
	return nil
}

// convertSNMPValue converts the SNMP variable to the specified data type
func convertSNMPValue(variable gosnmp.SnmpPDU, dataType string) (interface{}, error) {
	switch dataType {
	case "string":
		return variable.Value.(string), nil // Convert to string
	case "int":
		switch v := variable.Value.(type) {
		case int:
			return v, nil // Convert to int
		case int32:
			return int(v), nil // Convert to int32
		case int64:
			return v, nil // Convert to int64
		default:
			return nil, fmt.Errorf("unexpected type for int conversion: %T", v)
		}
	case "float":
		switch v := variable.Value.(type) {
		case float32:
			return float64(v), nil // Convert to float32
		case float64:
			return v, nil // Convert to float64
		default:
			return nil, fmt.Errorf("unexpected type for float conversion: %T", v)
		}
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType) // Handle unsupported data types
	}
}

// isValidOID checks if the provided OID is valid
func isValidOID(oid string) bool {
	// Simple OID validation
	// Example: .1.3.6.1.2.1.1.1.0
	if len(oid) == 0 || (oid[0] != '.' && oid[0] != '1') {
		return false
	}

	parts := strings.Split(strings.TrimPrefix(oid, "."), ".") // Split OID into parts
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false // Return false if any part is not an integer
		}
	}
	return true // OID is valid
}
