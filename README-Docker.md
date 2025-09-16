# Go-Proxy Docker Deployment

## 🐳 Quick Start

### Build and Run with Docker
```bash
# Build image
make docker-build

# Run container
make docker-run

# View logs
make docker-logs
```

### Run with Docker Compose
```bash
# Start all services (proxy + example backends)
make docker-compose-up

# Stop all services
make docker-compose-down
```

## 📋 Container Details

### Image Specifications
- **Base Image**: Alpine Linux 3.18 (ultra-lightweight)
- **Go Version**: 1.21
- **Final Image Size**: ~15MB
- **Security**: Non-root user, minimal attack surface

### Exposed Ports
- **8080**: Main proxy port
- **8081**: Metrics and monitoring
- **8082**: Configuration API + Swagger UI

### Health Check
- **Endpoint**: http://localhost:8081/
- **Interval**: 30 seconds
- **Timeout**: 3 seconds
- **Retries**: 3

## 🔧 Configuration

### Volume Mounts
```bash
# Mount custom config
docker run -v /path/to/config.yaml:/app/config.yaml:ro go-proxy:latest
```

### Environment Variables
```bash
# Set production environment
docker run -e GO_ENV=production go-proxy:latest
```

## 🌐 Access Points

Once running, access:
- **Proxy**: http://localhost:8080
- **Metrics**: http://localhost:8081
- **API Docs**: http://localhost:8082/swagger
- **Config API**: http://localhost:8082/config

## 🏗️ Multi-Stage Build Benefits

1. **Small Final Image**: Only ~15MB vs ~300MB+ with full Go image
2. **Security**: No build tools in production image
3. **Performance**: Faster pulls and deployments
4. **Static Binary**: No external dependencies

## 🔒 Security Features

- ✅ **Non-root user** (appuser:1001)
- ✅ **Static binary** with no CGO
- ✅ **Minimal base image** (Alpine)
- ✅ **Read-only config** mount
- ✅ **Health checks** enabled
- ✅ **CA certificates** for HTTPS

## 📊 Example Usage

```bash
# Start the stack
docker-compose up -d

# Test proxy
curl http://localhost:8080

# Check health
curl http://localhost:8081

# View API docs
open http://localhost:8082/swagger

# Add server via API
curl -X POST http://localhost:8082/servers \
  -H "X-API-KEY: super-admin-key-999" \
  -H "Content-Type: application/json" \
  -d '{"backend_name":"web-servers","url":"http://backend1","weight":1}'
```