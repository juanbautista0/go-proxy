# Go-Proxy: Enterprise Load Balancer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)
[![API](https://img.shields.io/badge/API-REST-FF6B35?style=for-the-badge)](http://localhost:8082/swagger)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

**Go-Proxy** is a high-performance, enterprise-grade HTTP load balancer built with Domain-Driven Design (DDD) architecture. It provides intelligent traffic distribution, real-time configuration management, and automated scaling capabilities.

## 🚀 Key Features

- **🔄 Dynamic Configuration**: Real-time config updates without restarts
- **🧠 Smart Load Balancing**: 6 advanced algorithms with adaptive selection
- **📊 Intelligent Triggers**: Traffic and schedule-based auto-scaling
- **🔐 Secure API**: Multi-level authentication with admin controls
- **📈 Enterprise Monitoring**: Comprehensive metrics and health checks
- **🐳 Cloud-Ready**: Docker support with multi-stage builds
- **⚡ High Performance**: Circuit breakers, connection pooling, health checks

## 📋 Table of Contents

- [Architecture Overview](#-architecture-overview)
- [Load Balancing Algorithms](#-load-balancing-algorithms)
- [Configuration Management](#-configuration-management)
- [API Documentation](#-api-documentation)
- [Deployment Guide](#-deployment-guide)
- [Cloud Integration](#-cloud-integration)
- [Monitoring & Observability](#-monitoring--observability)
- [Development](#-development)

## 🏗️ Architecture Overview

Go-Proxy follows Domain-Driven Design principles with clean architecture separation:

```mermaid
graph TB
    subgraph "Client Layer"
        C1[Web Clients]
        C2[Mobile Apps]
        C3[API Clients]
    end
    
    subgraph "Go-Proxy Core"
        LB[Load Balancer<br/>Port 8080]
        API[Config API<br/>Port 8082]
        METRICS[Metrics Server<br/>Port 8081]
    end
    
    subgraph "Backend Services"
        B1[Backend 1<br/>:3001]
        B2[Backend 2<br/>:3002]
        B3[Backend N<br/>:300N]
    end
    
    subgraph "External Systems"
        CLOUD[Cloud Providers<br/>AWS/GCP/Azure]
        MONITOR[Monitoring<br/>Prometheus/Grafana]
    end
    
    C1 --> LB
    C2 --> LB
    C3 --> LB
    
    LB --> B1
    LB --> B2
    LB --> B3
    
    API --> CLOUD
    METRICS --> MONITOR
    
    style LB fill:#e1f5fe
    style API fill:#f3e5f5
    style METRICS fill:#e8f5e8
```

### System Components

```mermaid
graph LR
    subgraph "Domain Layer"
        E[Entities]
        I[Interfaces]
        V[Value Objects]
    end
    
    subgraph "Application Layer"
        PS[Proxy Service]
        TS[Trigger Service]
        SS[Smart Service]
    end
    
    subgraph "Infrastructure Layer"
        EB[Enterprise Balancer]
        CM[Config Manager]
        HC[Health Checker]
        AE[Action Executor]
    end
    
    PS --> E
    TS --> I
    SS --> V
    
    EB --> PS
    CM --> TS
    HC --> SS
    AE --> TS
    
    style E fill:#ffebee
    style PS fill:#e3f2fd
    style EB fill:#e8f5e8
```

## ⚖️ Load Balancing Algorithms

Go-Proxy implements 6 sophisticated load balancing algorithms with intelligent auto-selection:

### Algorithm Comparison

| Algorithm | Use Case | Pros | Cons |
|-----------|----------|------|------|
| **Adaptive Weighted** | General purpose | Self-optimizing, performance-aware | Higher CPU usage |
| **Least Connections** | Long-lived connections | Fair distribution | Connection counting overhead |
| **Response Time** | Latency-sensitive | Fastest response | Requires response time tracking |
| **Consistent Hash** | Session affinity | Sticky sessions, cache-friendly | Uneven distribution possible |
| **Power of Two** | High throughput | Low overhead, good distribution | Less precise than least connections |
| **Weighted Fair Queue** | Mixed workloads | QoS support, priority handling | Complex configuration |

### Algorithm Selection Flow

```mermaid
flowchart TD
    START[Request Arrives] --> EVAL[Evaluate Current Metrics]
    EVAL --> SCORE[Calculate Algorithm Scores]
    
    SCORE --> CHECK{Score Difference > Threshold?}
    CHECK -->|Yes| SWITCH[Switch Algorithm]
    CHECK -->|No| CURRENT[Use Current Algorithm]
    
    SWITCH --> SELECT[Select Best Server]
    CURRENT --> SELECT
    
    SELECT --> HEALTH{Server Healthy?}
    HEALTH -->|Yes| FORWARD[Forward Request]
    HEALTH -->|No| CIRCUIT{Circuit Open?}
    
    CIRCUIT -->|Yes| RETRY[Try Next Server]
    CIRCUIT -->|No| FORWARD
    
    RETRY --> SELECT
    FORWARD --> RESPONSE[Return Response]
    
    style EVAL fill:#e1f5fe
    style SWITCH fill:#fff3e0
    style HEALTH fill:#e8f5e8
```

## 🔧 Configuration Management

### Configuration Structure

```yaml
# Core proxy settings
proxy:
  port: 8080

# Backend server pools
backends:
  - name: "web-servers"
    servers:
      - url: "http://backend1:3001"
        weight: 3
        max_connections: 100
        health_check_endpoint: "/health"
    balance_mode: "adaptive_weighted"
    min_servers: 1
    max_servers: 10
    health_interval: "10s"
    circuit_breaker:
      enabled: true
      failure_threshold: 5
      recovery_timeout: "30s"

# Intelligent triggers
triggers:
  smart:
    enabled: true
    evaluation_interval: "5s"
    scale_up_score: 0.45
    scale_down_score: 0.15
  
  traffic:
    high_threshold: 50
    low_threshold: 5
    high_action: "scale_up"
    low_action: "scale_down"
  
  schedule:
    - time: "09:00"
      action: "morning_scale"

# Scaling actions
actions:
  scale_up:
    url: "http://localhost:8082/actions/scale_up"
    method: "POST"

# Security configuration
security:
  api_keys:
    - "dev-key-456"
    - "prod-key-789"
  admin_api_keys:
    - "super-admin-key-999"
```

### Configuration Hot-Reload

```mermaid
sequenceDiagram
    participant API as Config API
    participant CM as Config Manager
    participant LB as Load Balancer
    participant HC as Health Checker
    
    API->>CM: Update Configuration
    CM->>CM: Validate Config
    CM->>CM: Write to File
    CM->>LB: Notify Change
    CM->>HC: Update Health Checks
    
    LB->>LB: Update Server Pool
    LB->>LB: Recalculate Weights
    HC->>HC: Start/Stop Checks
    
    Note over API,HC: Zero-downtime configuration updates
```

## 📡 API Documentation

### Authentication Levels

```mermaid
graph TD
    subgraph "API Access Levels"
        PUBLIC[Public Endpoints<br/>No Auth Required]
        REGULAR[Regular API Keys<br/>Server Management]
        ADMIN[Admin API Keys<br/>Security Management]
    end
    
    subgraph "Endpoints"
        ACTIONS["/actions/*<br/>Scaling Actions"]
        CONFIG["/config<br/>Configuration"]
        SERVERS["/servers<br/>Server CRUD"]
        SECURITY["/security<br/>Key Management"]
        SWAGGER["/swagger<br/>Documentation"]
    end
    
    PUBLIC --> ACTIONS
    PUBLIC --> CONFIG
    PUBLIC --> SWAGGER
    
    REGULAR --> SERVERS
    REGULAR --> CONFIG
    
    ADMIN --> SECURITY
    ADMIN --> SERVERS
    ADMIN --> CONFIG
    
    style PUBLIC fill:#e8f5e8
    style REGULAR fill:#fff3e0
    style ADMIN fill:#ffebee
```

### API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/config` | GET | None | Get current configuration |
| `/config` | PUT | Regular | Update configuration |
| `/servers` | POST | Regular | Add backend server |
| `/servers` | PUT | Regular | Update server |
| `/servers` | DELETE | Regular | Remove server |
| `/security` | GET | Admin | View API keys |
| `/security` | PUT | Admin | Manage API keys |
| `/actions/scale_up` | POST | None | Scale up servers |
| `/actions/scale_down` | POST | None | Scale down servers |
| `/swagger` | GET | None | API documentation |

### Interactive Documentation

Access the full API documentation at: **http://localhost:8082/swagger**

## 🚀 Deployment Guide

### Quick Start

```bash
# Clone repository
git clone https://github.com/juanbautista0/go-proxy
cd go-proxy

# Local development
make run

# Docker deployment
make docker-compose-up

# Production build
make build
./bin/proxy
```

### Docker Deployment

#### Single Container
```bash
# Build optimized image (~15MB)
make docker-build

# Run with custom config
docker run -d \
  --name go-proxy \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  go-proxy:latest
```

#### Docker Compose Stack
```bash
# Start complete stack (proxy + backends)
make docker-compose-up

# View logs
make docker-logs

# Scale backends
docker-compose up -d --scale backend1=3
```

### Production Deployment

```mermaid
graph TB
    subgraph "Load Balancer Tier"
        LB1[Go-Proxy Instance 1]
        LB2[Go-Proxy Instance 2]
        LB3[Go-Proxy Instance N]
    end
    
    subgraph "Application Tier"
        APP1[App Server 1]
        APP2[App Server 2]
        APP3[App Server N]
    end
    
    subgraph "Data Tier"
        DB1[(Database 1)]
        DB2[(Database 2)]
        CACHE[(Redis Cache)]
    end
    
    subgraph "Monitoring"
        PROM[Prometheus]
        GRAF[Grafana]
        ALERT[AlertManager]
    end
    
    INTERNET[Internet] --> LB1
    INTERNET --> LB2
    INTERNET --> LB3
    
    LB1 --> APP1
    LB2 --> APP2
    LB3 --> APP3
    
    APP1 --> DB1
    APP2 --> DB2
    APP1 --> CACHE
    APP2 --> CACHE
    
    LB1 --> PROM
    LB2 --> PROM
    LB3 --> PROM
    
    PROM --> GRAF
    PROM --> ALERT
```


### Webhook Integration Flow

```mermaid
sequenceDiagram
    participant GP as Go-Proxy
    participant TS as Trigger System
    participant CP as Cloud Provider
    participant AS as Auto Scaler
    participant BS as Backend Servers
    
    GP->>TS: High Traffic Detected
    TS->>TS: Evaluate Smart Triggers
    TS->>CP: Execute Scale Up Action
    CP->>AS: Increase Desired Capacity
    AS->>BS: Launch New Instances
    BS->>GP: Register with Load Balancer
    GP->>GP: Update Server Pool
    
    Note over GP,BS: Automated horizontal scaling
```

## 📊 Monitoring & Observability

### Metrics Endpoints

| Endpoint | Description | Format |
|----------|-------------|---------|
| `/metrics` | Prometheus metrics | Text |
| `/health` | Health check | JSON |
| `/stats` | Real-time statistics | JSON |

### Key Metrics

```mermaid
graph LR
    subgraph "Traffic Metrics"
        RPS[Requests/Second]
        RT[Response Time]
        ER[Error Rate]
        AC[Active Connections]
    end
    
    subgraph "Server Metrics"
        SH[Server Health]
        SC[Server Connections]
        SR[Server Response Time]
        CB[Circuit Breaker State]
    end
    
    subgraph "System Metrics"
        CPU[CPU Usage]
        MEM[Memory Usage]
        GC[GC Statistics]
        GO[Goroutines]
    end
    
    RPS --> DASHBOARD[Monitoring Dashboard]
    RT --> DASHBOARD
    ER --> DASHBOARD
    SH --> DASHBOARD
    CPU --> DASHBOARD
    
    style DASHBOARD fill:#e1f5fe
```

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'go-proxy'
    static_configs:
      - targets: ['localhost:8081']
    scrape_interval: 15s
    metrics_path: /metrics
```

### Grafana Dashboard

Key panels to monitor:
- Request rate and response time
- Error rate and success rate
- Server health status
- Load balancing distribution
- Circuit breaker states
- Auto-scaling events

## 🛠️ Development

### Project Structure

```
go-proxy/
├── cmd/                    # Application entry points
│   └── main.go
├── internal/
│   ├── domain/            # Business logic and entities
│   │   ├── config.go
│   │   └── proxy.go
│   ├── application/       # Application services
│   │   ├── proxy_service.go
│   │   ├── trigger_service.go
│   │   └── smart_trigger_service.go
│   └── infrastructure/    # External concerns
│       ├── enterprise_balancer.go
│       ├── config_manager.go
│       ├── health_checker.go
│       └── action_executor.go
├── config.yaml           # Configuration file
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Development stack
└── README.md            # This file
```

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run with coverage
make test-coverage

# Build binary
make build

# Run locally
make run
```

### Testing

```bash
# Unit tests
make test-unit

# Integration tests
make test-integration

# Benchmarks
make bench

# Coverage report
make test-coverage
open coverage.html
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📈 Performance Characteristics

### Benchmarks

| Metric | Value | Notes |
|--------|-------|-------|
| **Throughput** | 50K+ RPS | Single instance |
| **Latency P50** | < 1ms | Local backends |
| **Latency P99** | < 10ms | Local backends |
| **Memory Usage** | ~50MB | Baseline |
| **CPU Usage** | ~5% | At 10K RPS |
| **Connections** | 10K+ | Concurrent |

### Scaling Limits

- **Backends per pool**: 1,000+
- **Concurrent connections**: 100K+
- **Configuration size**: 10MB+
- **Health check frequency**: 1s minimum
- **Trigger evaluation**: 100ms minimum

## 🔒 Security Features

- **Multi-level API authentication**
- **Rate limiting and DDoS protection**
- **Secure configuration management**
- **Health check validation**
- **Circuit breaker protection**
- **Non-root Docker execution**
- **Minimal attack surface**

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Support

- **Documentation**: [API Docs](http://localhost:8082/swagger)
- **Issues**: [GitHub Issues](https://github.com/juanbautista0/go-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/juanbautista0/go-proxy/discussions)

---

**Built with ❤️ using Go and modern cloud-native practices**