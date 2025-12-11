# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o modcal ./cmd/modcal

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/modcal .

# Copy example config as default (should be overridden with volume mount)
COPY example.config.yaml ./config.yaml

# Expose the default port
EXPOSE 8080

# Run the application
ENTRYPOINT ["./modcal"]
CMD ["-config", "/app/config.yaml"]
