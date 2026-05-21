# -- Stage 1: Build the Go binary --
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Initialize a module (required by newer Go versions even for standard library)
RUN go mod init homelab-ip-monitor

# Copy the source code
COPY main.go .

# Build a statically linked binary. 
# CGO_ENABLED=0 ensures it doesn't rely on host OS libraries.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-binary

# -- Stage 2: Create the minimal production image --
FROM alpine:latest

# We MUST install ca-certificates so the app can securely connect 
# to HTTPS endpoints (api.telegram.org and api.ipify.org)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app-binary ./ip-monitor

# Command to run when the container starts
CMD ["./ip-monitor"]