FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

# Build unified dataangel binary (requires CGO for sqlite3)
COPY cmd/dataangel/go.mod cmd/dataangel/go.sum /build/cmd/dataangel/
COPY internal/sidecar/go.mod internal/sidecar/go.sum /build/internal/sidecar/

WORKDIR /build/cmd/dataangel
RUN go mod download

COPY cmd/dataangel/. /build/cmd/dataangel/
COPY internal/sidecar/. /build/internal/sidecar/
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o dataangel .

# Build CLI (still useful for debug/ops)
COPY cmd/cli/go.mod /build/cmd/cli/
COPY cmd/data-guard-cli/go.mod /build/cmd/data-guard-cli/
COPY pkg/s3/go.mod /build/pkg/s3/

WORKDIR /build/cmd/cli
RUN go mod download

WORKDIR /build/cmd/data-guard-cli
COPY cmd/cli/. /build/cmd/cli/
COPY cmd/data-guard-cli/. /build/cmd/data-guard-cli/
COPY pkg/s3/. /build/pkg/s3/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cli .

FROM alpine:latest

RUN apk update && apk --no-cache add ca-certificates rclone curl && \
    curl -L https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.tar.gz | tar xz -C /usr/local/bin litestream

RUN addgroup -g 1000 dataguard && \
    adduser -u 1000 -G dataguard -s /bin/sh -D dataguard

WORKDIR /home/dataguard

COPY --from=builder --chown=dataguard:dataguard /build/cmd/dataangel/dataangel .
COPY --from=builder --chown=dataguard:dataguard /build/cmd/data-guard-cli/cli .

USER dataguard

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["./dataangel"]
