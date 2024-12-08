# Use the official Go image as a base
FROM golang:1.21.6

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the default port (this is optional and doesn't need to match the dynamic port)
EXPOSE 8080

# Run the application
CMD ["./main"]
