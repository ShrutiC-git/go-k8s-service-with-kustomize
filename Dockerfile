# Stage 1: Build the application binary
FROM golang:1.22 as builder

WORKDIR /app

# Copy module files and download dependencies first.
# This leverages Docker's layer caching. Dependencies will only be
# re-downloaded if go.mod or go.sum changes.
COPY go.mod go.sum ./
RUN go fmt -s .
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go app, creating a statically linked binary for Linux.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o checkout-service .

# ---

# Stage 2: Create the final, minimal image
FROM alpine:latest

# Copy the compiled binary from the 'builder' stage.
COPY --from=builder /app/checkout-service .

# Tell Docker that the container listens on this port at runtime.
EXPOSE 8080

# Command to run the executable when the container starts.
CMD ["./checkout-service"]