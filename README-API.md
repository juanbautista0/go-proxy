# Go-Proxy Configuration API

## 📖 Swagger Documentation

**Access interactive documentation:**
- **Swagger UI**: http://localhost:8082/swagger
- **YAML Spec**: http://localhost:8082/api-docs.yaml

## 🔐 Authentication

Protected endpoints require `X-API-KEY` header:

```bash
curl -H "X-API-KEY: YOUR_API_KEY" http://localhost:8082/servers
```

**API Keys:**
- Contact administrator to get a valid key
- Keys are confidential and not shown in documentation

## 🚀 Usage Examples

### Add Server
```bash
curl -X POST http://localhost:8082/servers \
  -H "X-API-KEY: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "backend_name": "web-servers",
    "url": "http://localhost:3004",
    "weight": 2,
    "max_connections": 150,
    "health_check_endpoint": "/health"
  }'
```

### Update Server
```bash
curl -X PUT http://localhost:8082/servers \
  -H "X-API-KEY: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "backend_name": "web-servers",
    "old_url": "http://localhost:3004",
    "url": "http://localhost:3005",
    "weight": 3,
    "max_connections": 200,
    "health_check_endpoint": "/status"
  }'
```

### Remove Server
```bash
curl -X DELETE http://localhost:8082/servers \
  -H "X-API-KEY: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "backend_name": "web-servers",
    "server_url": "http://localhost:3004"
  }'
```

### Scaling Actions
```bash
# Scale Up (no authentication)
curl -X POST http://localhost:8082/actions/scale_up

# Scale Down (no authentication)
curl -X POST http://localhost:8082/actions/scale_down
```

## ⚠️ Limits and Validations

- **Minimum servers**: 1 (configurable in `min_servers`)
- **Maximum servers**: 10 (configurable in `max_servers`)
- **Proxy port**: Not modifiable via API for security
- **Automatic validation** on all operations
- **Descriptive error responses** with appropriate HTTP codes

## 📊 Response Codes

- `200` - Successful operation
- `201` - Resource created
- `400` - Limits reached or invalid data
- `401` - API Key required or invalid
- `404` - Resource not found
- `500` - Internal server error