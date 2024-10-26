# Stage 1: Build the Go app
FROM golang:1.20-alpine AS builder

# Set environment variables
ENV GO111MODULE=on

# Install necessary packages
RUN apk update && apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o main .

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
