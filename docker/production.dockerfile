FROM golang:1.23

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 is important for static binaries if you were to switch to distroless later,
# but less critical for golang:bookworm which has libc. Still good practice for cross-platform.
# -o specifies the output path and filename for the executable
# ./... ensures all packages in the current directory and its subdirectories are built
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/app ./

# Expose any ports your application listens on (optional, but good practice)
EXPOSE 8080

# Command to run the application
CMD ["/app/app"]
