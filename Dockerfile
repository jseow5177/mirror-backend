FROM golang:1.22.7-alpine AS builder

WORKDIR /app

# Copy the Go source code
COPY . ./

# Ensure build script is executable
RUN chmod +x scripts/build.sh

# Run the build script
RUN ./scripts/build.sh

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/mirror-backend .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./mirror-backend"]
