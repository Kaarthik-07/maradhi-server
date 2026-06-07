# ---- Build Stage ----
FROM golang:1.22-alpine AS builder

# Install git (needed for go mod download with some deps)
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies first (faster rebuilds)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# ---- Run Stage ----
FROM alpine:3.19

# Add ca-certificates for HTTPS calls (e.g. Supabase, Claude API)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy only the binary from the build stage
COPY --from=builder /app/server .

# Render injects PORT env var — your server should read this
# Make sure your main.go uses: os.Getenv("PORT") with a fallback
EXPOSE 8080

CMD ["./server"]
