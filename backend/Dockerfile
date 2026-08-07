
FROM golang:1.25-alpine

WORKDIR /app

# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest

# Copy go.mod
COPY go.mod .

# Copy source code
COPY . .

# Use air instead of direct binary execution
CMD ["air", "-c", ".air.toml"]
