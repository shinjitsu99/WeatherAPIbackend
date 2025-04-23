# syntax=docker/dockerfile:1
FROM golang:1.24-alpine

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o server main.go

# Expose the port your app listens on
EXPOSE 8081

# Start the app
CMD ["./server"]
