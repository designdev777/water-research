# Multi-stage build for smaller production image
FROM golang:1.22-alpine AS builder

# Install build dependencies for SQLite (CGO required)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLite support
# -s -w strip debug symbols to reduce binary size
RUN CGO_ENABLED=1 go build -tags netgo -ldflags '-s -w' -o water-research .

# Final stage - minimal production image
FROM alpine:latest

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates sqlite-libs tzdata

WORKDIR /app

# Create directory for static files and proposals
RUN mkdir -p static/proposals

# Copy binary from builder
COPY --from=builder /app/water-research .

# Copy template files
COPY --from=builder /app/templates ./templates

# Copy static files if any
COPY --from=builder /app/static ./static

# Expose port (Render will provide PORT env variable)
EXPOSE 8080

# Run the application
CMD ["./water-research"]