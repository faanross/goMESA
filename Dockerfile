FROM golang:1.20-alpine as builder

# Install build dependencies
RUN apk add --no-cache git make gcc libc-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make server

# Create runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache libpcap libpcap-dev tzdata ca-certificates

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/mesa-server /app/mesa-server

# Expose NTP port
EXPOSE 123/udp

# Set entrypoint
ENTRYPOINT ["/app/mesa-server"]