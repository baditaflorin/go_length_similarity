FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk --no-cache add git ca-certificates build-base

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/similarity-server \
    ./cmd/server

# Create a minimal production image
FROM alpine:3.18

# Install required runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/similarity-server /app/

# Expose the application port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/similarity-server"]