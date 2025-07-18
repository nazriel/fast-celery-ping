# ----------- Stage 1: Build -----------
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH
ARG TARGETOS
ARG VERSION=dev
ARG BUILD_TIME

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w \
    -X 'fast-celery-ping/cmd.Version=${VERSION}' \
    -X 'fast-celery-ping/cmd.BuildTime=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}'" \
    -o fast-celery-ping .

# ----------- Stage 2: Minimal runtime -----------
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/fast-celery-ping ./fast-celery-ping

ENTRYPOINT ["/app/fast-celery-ping"]

CMD ["--help"]
