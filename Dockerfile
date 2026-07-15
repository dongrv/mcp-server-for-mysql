FROM golang:1.24-alpine AS builder

RUN apk add --no-cache ca-certificates git
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mcp-database ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -g 1000 -S appuser \
    && adduser -u 1000 -S appuser -G appuser

WORKDIR /app
COPY --from=builder /out/mcp-database ./mcp-database
# This is documentation only. Mount a real config.yaml and provide its DSN
# environment references at runtime.
COPY --from=builder /app/config.example.yaml ./config.example.yaml

RUN chown -R appuser:appuser /app
USER appuser

ENTRYPOINT ["./mcp-database"]
