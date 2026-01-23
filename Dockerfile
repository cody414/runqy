# Build stage
FROM golang:1.24-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /build

# Copy go mod files first for better caching
COPY app/go.mod app/go.sum ./
RUN go mod download

# Copy source code
COPY app/ .

# Build with version info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o /runqy .

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -h /app runqy
USER runqy
WORKDIR /app

# Copy binary from builder
COPY --from=builder /runqy /usr/local/bin/runqy

# Copy deployment examples (optional, can be mounted)
COPY --chown=runqy:runqy deployment/ /app/deployment/

EXPOSE 3000

ENTRYPOINT ["runqy"]
CMD ["serve"]
