# Use the official Go image as a base
FROM golang:1.21.6

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Copy credentials.json into the container
COPY credentials.json /app/credentials.json

# List files in /app to verify credentials.json is present
RUN ls -al /app

# Build the Go application
RUN go build -o main .

# Expose the port your app runs on
EXPOSE 8080

# Run the application
CMD ["./main"]