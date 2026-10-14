# Build Stage
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

# Optimize caching by downloading dependencies first
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile static binary (CGO_ENABLED=0) optimized for size (-w -s)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o cleantrace ./cmd/cleantrace

# Final Stage (Distroless for maximum security and minimal attack surface)
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/cleantrace .

# Run as an unprivileged user
USER 65532:65532

EXPOSE 8080

CMD ["./cleantrace"]