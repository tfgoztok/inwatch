# InWatch Data Puller

InWatch Data Puller is a highly configurable data collection service designed to pull metrics from various network devices. This is the first beta version that currently supports SNMP protocol and uses PostgreSQL for configuration management.

## Features

- **Protocol Support**
  - SNMP (v2c) implemented
  - Extensible architecture for future protocol implementations (TCP, Modbus, REST planned)

- **Dynamic Configuration**
  - Real-time configuration changes through PostgreSQL
  - Device management
  - Query management
  - Protocol-specific configurations

- **Robust Data Collection**
  - Configurable polling intervals
  - Built-in rate limiting
  - Automatic retry mechanism
  - Worker pool for concurrent data collection

## Architecture

The service consists of several key components:

- **Collector**: Manages devices and orchestrates data collection
- **Protocol Handlers**: Implements protocol-specific data collection logic
- **Config Manager**: Handles dynamic configuration updates
- **Work Queue**: Manages scheduled data collection tasks
- **Database**: Stores device configurations and collection parameters

## Prerequisites

- Docker and Docker Compose
- PostgreSQL 15 or later
- Go 1.21 or later (for development)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/tfgoztok/inwatch.git
cd inwatch
```

2. Start the services using Docker Compose:
```bash
docker-compose up -d
```

The service will automatically:
- Set up the PostgreSQL database
- Run necessary migrations
- Start the data collection service

## Configuration

### Database Schema

The service uses three main tables:

- `devices`: Stores device information
- `device_queries`: Defines what data to collect from each device
- `device_configs`: Stores protocol-specific configuration for devices

### Environment Variables

```
DB_HOST=postgres
DB_PORT=5432
DB_USER=inwatch
DB_PASSWORD=inwatch123
DB_NAME=inwatch
```

## Development

### Project Structure

```
├── cmd/
│   └── main.go           # Application entry point
├── internal/
│   ├── collector/        # Data collection orchestration
│   ├── config/          # Configuration management
│   ├── database/        # Database operations and migrations
│   ├── model/           # Data models
│   ├── protocol/        # Protocol implementations
│   │   └── snmp/       # SNMP protocol handler
│   └── queue/           # Work queue management
├── docker-compose.yml
└── Dockerfile
```

### Building

```bash
go build -o data_puller ./cmd/main.go
```

### Running Tests

```bash
# Tests to be implemented
```

## Current Limitations

- Only SNMP v2c protocol is implemented
- No data persistence layer for collected metrics
- No authentication mechanism
- No tests implemented yet**
- Basic error handling and logging
- Limited configuration validation

## Roadmap

- [ ] Add support for additional protocols (TCP, Modbus, REST)
- [ ] Implement data persistence for collected metrics
- [ ] Add authentication and authorization
- [ ] Implement comprehensive testing
- [ ] Add metrics and monitoring
- [ ] Enhance error handling and logging
- [ ] Add configuration validation
- [ ] Implement secure protocol versions (SNMPv3)

## Current Arthitecture
flowchart TD
    subgraph Docker ["Docker Environment"]
        subgraph DataPuller ["Data Puller Service"]
            Collector["Collector"]
            WorkQueue["Work Queue"]
            RateLimit["Rate Limiter"]
            RetryPolicy["Retry Policy"]
            ConfigMgr["Config Manager"]
            
            subgraph Protocols ["Protocol Handlers"]
                SNMP["SNMP Handler"]
                Future["Future Handlers\n(TCP/Modbus/REST)"]
                style Future fill:#f5f5f5,stroke-dash: 5
            end
        end
        
        subgraph Database ["PostgreSQL Database"]
            Devices[("Devices")]
            Queries[("Device Queries")]
            Configs[("Device Configs")]
            
            Devices --> Queries
            Devices --> Configs
        end
        
        subgraph Target ["Target Devices"]
            SNMPDev["SNMP Devices"]
            OtherDev["Other Devices"]
            style OtherDev fill:#f5f5f5,stroke-dash: 5
        end
    end
    
    %% Data flow connections
    ConfigMgr -->|"Watch Changes"| Database
    ConfigMgr -->|"Update Config"| Collector
    
    Collector -->|"Schedule Tasks"| WorkQueue
    WorkQueue -->|"Execute Tasks"| RateLimit
    RateLimit -->|"Control Flow"| RetryPolicy
    RetryPolicy -->|"Handle Retries"| SNMP
    
    SNMP -->|"Poll Data"| SNMPDev
    Future -->|"Future Connection"| OtherDev
    
    %% Notifications flow
    Database -->|"Notify Changes"| ConfigMgr
    
    %% Styling
    classDef primary fill:#d1eaff,stroke:#2d7edb
    classDef secondary fill:#fff3cd,stroke:#f5b400
    classDef database fill:#d4edda,stroke:#28a745
    classDef external fill:#e2e3e5,stroke:#383d41
    
    class Collector,ConfigMgr primary
    class WorkQueue,RateLimit,RetryPolicy secondary
    class Database,Devices,Queries,Configs database
    class SNMPDev,OtherDev external