# Multi-stage build Dockerfile for distr-comp project

# Build stage
FROM golang:1.23.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache make git

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Orchestrator stage
FROM alpine:3.17 AS orchestrator

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/build/orchestrator /app/orchestrator

# Expose the orchestrator port
EXPOSE 8080

# Set environment variables for operation times (in milliseconds)
ENV TIME_ADDITION=1 \
    TIME_SUBTRACTION=1 \
    TIME_MULTIPLICATION=1 \
    TIME_DIVISION=1

# Run the orchestrator
CMD ["/app/orchestrator"]

# Agent stage
FROM alpine:3.17 AS agent

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/build/agent /app/agent

# Set environment variables
ENV COMPUTING_POWER=10 \
    ORCHESTRATOR_URL="http://orchestrator:8080"

# Run the agent
CMD ["/app/agent"]