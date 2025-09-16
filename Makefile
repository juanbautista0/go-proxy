build:
	go build -o bin/proxy cmd/main.go

run:
	go run cmd/main.go

run-config:
	go run cmd/main.go $(CONFIG)

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@docker --version || (echo "Docker not installed or not running" && exit 1)
	docker build --no-cache -t go-proxy:latest .

docker-run: docker-build docker-stop
	docker run -d --name go-proxy \
		-p 8080:8080 -p 8081:8081 -p 8082:8082 \
		-v $(PWD)/config.yaml:/app/config.yaml:ro \
		go-proxy:latest

docker-stop:
	-docker stop go-proxy 2>/dev/null || true
	-docker rm go-proxy 2>/dev/null || true

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-logs:
	docker logs -f go-proxy

docker-clean:
	docker-compose down
	-docker stop go-proxy 2>/dev/null || true
	-docker rm go-proxy 2>/dev/null || true
	-docker rmi go-proxy:latest 2>/dev/null || true

test:
	go test ./internal/... ./cmd/... -v

test-coverage:
	go test ./internal/... ./cmd/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-unit:
	go test ./internal/... -v

test-integration:
	go test ./internal/... ./cmd/... -tags=integration -v

bench:
	go test ./internal/... ./cmd/... -bench=. -benchmem

clean:
	rm -rf bin/

.PHONY: build run test clean docker-build docker-run docker-stop docker-clean docker-compose-up docker-compose-down