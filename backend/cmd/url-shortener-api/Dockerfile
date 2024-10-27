# Stage 1: Build the Go app
FROM golang:1.23.1-alpine AS builder

# Set environment variables
ENV GO111MODULE=on

# Install necessary packages
RUN apk update && apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum from the correct subdirectory
COPY cmd/url-shortener-api/go.mod cmd/url-shortener-api/go.sum ./

# Download dependencies
RUN go mod download

# Build the application
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/url-shortener-api

# Stage 2: Run the Go app in a minimal container
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose the port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
