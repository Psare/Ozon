# Use the golang base image for building the Go application
FROM golang:latest AS builder

# Set the working directory
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download the Go dependencies
RUN go mod download

# Copy the source code to the container
COPY . .

# Build the Go application
RUN go build -o main .

# Copy the built Go application from the builder stage
COPY --from=builder /app/main /app/main

# Set the entrypoint command to run the Go application
CMD ["/app/main"]
