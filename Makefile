# Makefile for the distr-comp project

# Variables
APP_NAME = distr-comp
ORCHESTRATOR_BIN = orchestrator
AGENT_BIN = agent
DOCKER_IMAGE = distr-comp
DOCKER_TAG = latest

# Go commands
GO = go
GO_BUILD = $(GO) build
GO_TEST = $(GO) test
GO_CLEAN = $(GO) clean
GO_FMT = $(GO) fmt
GO_VET = $(GO) vet
GO_MOD = $(GO) mod

# Directories
CMD_DIR = ./cmd
INTERNAL_DIR = ./internal
BUILD_DIR = ./build

# Detect OS for cross-platform compatibility
ifeq ($(OS),Windows_NT)
	RM = if exist $(BUILD_DIR) rd /s /q $(BUILD_DIR)
	MKDIR = if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	SEP = \\
else
	RM = rm -rf $(BUILD_DIR)
	MKDIR = mkdir -p $(BUILD_DIR)
	SEP = /
endif

# Targets
.PHONY: all build clean test fmt vet docker-build docker-run docker-push

all: clean fmt vet test build

# Build all binaries
build: build-orchestrator build-agent

build-orchestrator:
	@echo "Building orchestrator..."
	@$(MKDIR)
	$(GO_BUILD) -o $(BUILD_DIR)$(SEP)$(ORCHESTRATOR_BIN)$(if $(filter $(OS),Windows_NT),.exe,) $(CMD_DIR)/orchestrator/main.go

build-agent:
	@echo "Building agent..."
	@$(MKDIR)
	$(GO_BUILD) -o $(BUILD_DIR)$(SEP)$(AGENT_BIN)$(if $(filter $(OS),Windows_NT),.exe,) $(CMD_DIR)/agent/main.go

# Clean
clean:
	@echo "Cleaning..."
	@-$(RM)
	$(GO_CLEAN) ./...

# Testing
test:
	@echo "Running tests..."
	$(GO_TEST) ./...

# Code formatting
fmt:
	@echo "Formatting code..."
	$(GO_FMT) ./...

# Static code analysis
vet:
	@echo "Vetting code..."
	$(GO_VET) ./...

# Dependency check
deps:
	@echo "Checking dependencies..."
	$(GO_MOD) tidy
	$(GO_MOD) verify