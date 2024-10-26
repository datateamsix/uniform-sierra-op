# backend/Dockerfile 

#Stage 1: Build the GO application
FROM golang:1.22.3-alpine as builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

#Install the necessary packages and dependencies
RUN apk update && apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

#Copy the go mod and sum files
COPY go.mod go.sum ./

#Download the dependencies
RUN go mod download 

#Copy the source code into the container
COPY . .

#Build the GO Application 
RUN go build -o main ./cmd/url-shortener-api/main.go

#Stage 2: Run Go App in minimal container
FROM alpine:latest

#Install the necessary packages and dependencies
RUN apk update && apk add --no-cache ca-certificates

#Set the working directory 
WORKDIR /root/

# Copy the binary from the builder stage 
COPY --from=builder /app/main .

#Expose the port that the application listens on
EXPOSE 8080

#Command to run the executable
CMD ["./main"]  