# Go A2A Makefile

# Variables
BINARY_DIR := bin
SERVER_BINARY := $(BINARY_DIR)/a2a-server
CLIENT_BINARY := $(BINARY_DIR)/a2a-client
PLUGIN_DIR := plugins
PLUGIN_BINARY := $(PLUGIN_DIR)/echo_plugin.so
CONFIG_DIR := config

# Go build flags
GOFLAGS := -v

# Default target
.PHONY: all
all: build

# Create directories
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

$(CONFIG_DIR):
	mkdir -p $(CONFIG_DIR)

# Build targets
.PHONY: build
build: $(BINARY_DIR) build-server build-client build-plugin

.PHONY: build-server
build-server: $(SERVER_BINARY)

.PHONY: build-client
build-client: $(CLIENT_BINARY)

.PHONY: build-plugin
build-plugin: $(PLUGIN_BINARY)

$(SERVER_BINARY): cmd/a2a-server/main.go
	go build $(GOFLAGS) -o $(SERVER_BINARY) ./cmd/a2a-server

$(CLIENT_BINARY): cmd/a2a-client/main.go
	go build $(GOFLAGS) -o $(CLIENT_BINARY) ./cmd/a2a-client

$(PLUGIN_BINARY): plugins/echo_plugin.go
	go build $(GOFLAGS) -buildmode=plugin -o $(PLUGIN_BINARY) ./plugins/echo_plugin.go

# Run targets
.PHONY: run-server
run-server: build-server
	$(SERVER_BINARY) --config $(CONFIG_DIR)/server.json

.PHONY: run-server-ollama
run-server-ollama: build-server
	PROVIDER=ollama MODEL=llama3 OPENAI_API_KEY=ollama $(SERVER_BINARY) --config $(CONFIG_DIR)/server.json

.PHONY: run-client
run-client: build-client
	$(CLIENT_BINARY) --config $(CONFIG_DIR)/client.json card

# Docker targets
.PHONY: docker-build
docker-build:
	docker build -t go-a2a .

.PHONY: docker-run-server
docker-run-server: docker-build
	docker run -p 8080:8080 -v $(PWD)/$(CONFIG_DIR):/app/config -v $(PWD)/$(PLUGIN_DIR):/app/plugins go-a2a

.PHONY: docker-run-client
docker-run-client: docker-build
	docker run --entrypoint /app/bin/a2a-client go-a2a --url http://localhost:8080 card

.PHONY: docker-compose-up
docker-compose-up:
	docker-compose up

.PHONY: docker-compose-down
docker-compose-down:
	docker-compose down

# Clean targets
.PHONY: clean
clean:
	rm -rf $(BINARY_DIR)
	rm -f $(PLUGIN_BINARY)

# Test targets
.PHONY: test
test:
	go test -v ./...

# Help target
.PHONY: help
help:
	@echo "Go A2A Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all            Build server and client binaries (default)"
	@echo "  build          Build server and client binaries"
	@echo "  build-server   Build server binary"
	@echo "  build-client   Build client binary"
	@echo "  build-plugin   Build plugin binary"
	@echo "  run-server     Run server"
	@echo "  run-server-ollama Run server with Ollama LLM"
	@echo "  run-client     Run client"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run-server Run server in Docker"
	@echo "  docker-run-client Run client in Docker"
	@echo "  docker-compose-up Run server and client with Docker Compose"
	@echo "  docker-compose-down Stop Docker Compose services"
	@echo "  clean          Remove build artifacts"
	@echo "  test           Run tests"
	@echo "  help           Show this help message"
