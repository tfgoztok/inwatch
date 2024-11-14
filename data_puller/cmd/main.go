package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"data_puller/internal/collector"
	"data_puller/internal/config"
	"data_puller/internal/model"
	"data_puller/internal/protocol/snmp"
)

func main() {
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation on function exit

	// Initialize config manager with database connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	configManager, err := config.NewConfigManager(dsn)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}
	defer configManager.Close()

	if err := configManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize config manager: %v", err)
	}

	// Create collector with specified queue size and number of workers
	collector := collector.NewCollector(1000, 5) // 1000 queue size, 5 workers

	// Register SNMP handler with the collector
	if err := collector.RegisterHandler("snmp", snmp.NewSNMPHandler()); err != nil {
		log.Fatalf("Failed to register SNMP handler: %v", err)
	}

	// Start the collector to begin processing
	if err := collector.Start(ctx); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}

	// Load initial devices from the config manager
	devices, err := configManager.GetDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to get devices: %v", err)
	}

	// Configure initial devices by retrieving their queries and configs
	for _, device := range devices {
		// Get device queries
		queries, err := configManager.GetDeviceQueries(ctx, device.ID)
		if err != nil {
			log.Printf("Failed to get queries for device %d: %v", device.ID, err)
			continue
		}

		// Get device configs
		configs, err := configManager.GetDeviceConfig(ctx, device.ID)
		if err != nil {
			log.Printf("Failed to get configs for device %d: %v", device.ID, err)
			continue
		}

		// Add device to collector
		if err := collector.AddDevice(device, configs); err != nil {
			log.Printf("Failed to add device %d: %v", device.ID, err)
			continue
		}

		// Schedule queries
		for _, query := range queries {
			if err := collector.ScheduleQuery(ctx, device.ID, query); err != nil {
				log.Printf("Failed to schedule query for device %d: %v", device.ID, err)
			}
		}
	}

	// Watch for configuration changes in a separate goroutine
	changes, err := configManager.WatchChanges(ctx)
	if err != nil {
		log.Fatalf("Failed to watch config changes: %v", err)
	}

	// Handle config changes
	go func() {
		for change := range changes {
			switch change.Type {
			case "device_added":
				device := change.Data.(*model.Device)
				configs, _ := configManager.GetDeviceConfig(ctx, device.ID)
				collector.AddDevice(device, configs)

			case "device_updated":
				device := change.Data.(*model.Device)
				configs, _ := configManager.GetDeviceConfig(ctx, device.ID)
				collector.UpdateDevice(device, configs)

			case "device_removed":
				collector.RemoveDevice(change.DeviceID)

			case "query_updated":
				query := change.Data.(*model.DeviceQuery)
				collector.ScheduleQuery(ctx, query.DeviceID, query)
			}
		}
	}()

	// Handle shutdown gracefully on receiving termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel main context to stop new work
	cancel()

	// Stop collector
	if err := collector.Stop(); err != nil {
		log.Printf("Error stopping collector: %v", err)
	}

	// Wait for shutdown context
	<-shutdownCtx.Done()
	log.Println("Shutdown complete")
}
