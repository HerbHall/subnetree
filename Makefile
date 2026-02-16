.PHONY: build build-server build-scout build-dashboard dev-dashboard lint-dashboard test test-race test-coverage lint run-server run-scout proto swagger clean license-check ai-review ai-test ai-doc docker-build docker-test docker-clean docker-scout docker-scout-full docker-qc docker-qc-down docker-qc-smoke

# Binary names
SERVER_BIN=subnetree
SCOUT_BIN=scout

# Frontend
PNPM=pnpm
WEB_DIR=web

# Version injection
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
VERSION_PKG = github.com/HerbHall/subnetree/internal/version

# Build flags
LDFLAGS=-ldflags "-s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).GitCommit=$(COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(DATE)"

# Full build: frontend first, then Go binaries
build: build-dashboard build-server build-scout

build-server:
	go build $(LDFLAGS) -o bin/$(SERVER_BIN) ./cmd/subnetree/

build-scout:
	go build $(LDFLAGS) -o bin/$(SCOUT_BIN) ./cmd/scout/

# Frontend targets
build-dashboard:
	cd $(WEB_DIR) && $(PNPM) install --frozen-lockfile && $(PNPM) run build
	rm -rf internal/dashboard/dist
	cp -r $(WEB_DIR)/dist internal/dashboard/dist

dev-dashboard:
	cd $(WEB_DIR) && $(PNPM) install && $(PNPM) run dev

lint-dashboard:
	cd $(WEB_DIR) && $(PNPM) run lint && $(PNPM) run type-check

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

run-server: build-server
	./bin/$(SERVER_BIN)

run-scout: build-scout
	./bin/$(SCOUT_BIN)

proto:
	protoc -I. -I$(shell go env GOPATH)/include \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/v1/*.proto

swagger:
	@which swag > /dev/null 2>&1 || (echo "swag not found. Install: go install github.com/swaggo/swag/cmd/swag@latest" && exit 1)
	swag init -g cmd/subnetree/main.go -o api/swagger --parseDependency --parseInternal

# Allowed licenses for dependencies (BSL 1.1 compatible)
ALLOWED_LICENSES=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0

license-check:
	@echo "Checking dependency licenses..."
	@go-licenses check ./... --allowed_licenses=$(ALLOWED_LICENSES) \
		|| (echo "ERROR: Incompatible license detected. See go-licenses output above." && exit 1)
	@echo "All dependency licenses are compatible."

license-report:
	@go-licenses report ./... --template=csv 2>/dev/null || go-licenses csv ./... 2>/dev/null

# AI dev tools (requires Ollama on localhost:11434)
OLLAMA_HOST ?= http://127.0.0.1:11434

ai-review:
	@bash tools/ai-review.sh

ai-review-all:
	@bash tools/ai-review.sh --all

ai-test:
	@test -n "$(FILE)" || (echo "Usage: make ai-test FILE=path/to/file.go" && exit 1)
	@bash tools/ai-test.sh $(FILE)

ai-doc:
	@test -n "$(FILE)" || (echo "Usage: make ai-doc FILE=path/to/file.go" && exit 1)
	@bash tools/ai-doc.sh $(FILE)

# Docker local testing
DOCKER_IMAGE ?= subnetree:local
DOCKER_TEST_PORT ?= 19998
DOCKER_TEST_CONTAINER ?= subnetree-test

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(DATE) \
		-t $(DOCKER_IMAGE) .

docker-test: docker-build
	@bash tools/docker-smoke-test.sh $(DOCKER_IMAGE) $(DOCKER_TEST_PORT) $(DOCKER_TEST_CONTAINER)

docker-clean:
	-docker rm -f $(DOCKER_TEST_CONTAINER) 2>/dev/null
	-docker rmi $(DOCKER_IMAGE) 2>/dev/null
	-docker volume rm subnetree-test-data 2>/dev/null

docker-scout: docker-build
	docker scout quickview $(DOCKER_IMAGE)
	docker scout cves $(DOCKER_IMAGE) --only-severity critical,high

docker-scout-full: docker-build
	docker scout cves $(DOCKER_IMAGE)
	docker scout recommendations $(DOCKER_IMAGE)

# Pre-release QC testing (local build + seed data)
QC_COMPOSE=docker compose -f docker-compose.qc.yml

docker-qc: ## Build from source and run with seed data for manual QC
	$(QC_COMPOSE) down -v 2>/dev/null || true
	$(QC_COMPOSE) up --build

docker-qc-down: ## Stop and clean QC environment
	$(QC_COMPOSE) down -v

docker-qc-smoke: ## Build, start detached, wait for healthy, run smoke tests
	$(QC_COMPOSE) down -v 2>/dev/null || true
	$(QC_COMPOSE) up --build -d
	@echo "Waiting for container to be healthy..."
	@for i in $$(seq 1 60); do \
		if docker inspect --format='{{.State.Health.Status}}' subnetree-qc 2>/dev/null | grep -q healthy; then \
			echo "Container healthy after $${i}s"; \
			break; \
		fi; \
		if [ $$i -eq 60 ]; then \
			echo "ERROR: Container not healthy after 60s"; \
			$(QC_COMPOSE) logs; \
			$(QC_COMPOSE) down -v; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	@bash tools/docker-smoke-test.sh subnetree-qc:latest 8080 subnetree-qc-smoke || true
	$(QC_COMPOSE) down -v

clean:
	rm -rf bin/ $(WEB_DIR)/dist $(WEB_DIR)/node_modules/.cache internal/dashboard/dist
	go clean
