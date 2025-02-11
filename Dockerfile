FROM golang:1.22.7-alpine AS builder

WORKDIR /app

# Copy all files
COPY . ./

# Ensure build script has execution permissions
RUN chmod +x scripts/build.sh && /bin/sh scripts/build.sh
RUN chmod +x scripts/build-job.sh && /bin/sh scripts/build-job.sh

# Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/mirror-backend .
COPY --from=builder /app/bin/mirror-backend-job .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./mirror-backend"]
