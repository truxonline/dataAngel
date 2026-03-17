FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Build init-container
COPY cmd/init/go.mod /build/cmd/init/
COPY internal/restore/go.mod /build/internal/restore/
COPY pkg/s3/go.mod /build/pkg/s3/

WORKDIR /build/internal/restore
RUN go mod download

WORKDIR /build/cmd/init
COPY cmd/init/. /build/cmd/init/
COPY internal/restore/. /build/internal/restore/
COPY pkg/s3/. /build/pkg/s3/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o init .

# Build CLI
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

# Build sidecar
COPY cmd/sidecar/go.mod cmd/sidecar/go.sum* /build/cmd/sidecar/
COPY internal/sidecar/go.mod internal/sidecar/go.sum /build/internal/sidecar/

WORKDIR /build/cmd/sidecar
RUN go mod download

COPY cmd/sidecar/. /build/cmd/sidecar/
COPY internal/sidecar/. /build/internal/sidecar/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sidecar .

FROM alpine:latest

RUN apk update && apk --no-cache add ca-certificates rclone curl && \
    curl -L https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.tar.gz | tar xz -C /usr/local/bin litestream

RUN addgroup -g 1000 dataguard && \
    adduser -u 1000 -G dataguard -s /bin/sh -D dataguard

WORKDIR /home/dataguard

COPY --from=builder --chown=dataguard:dataguard /build/cmd/init/init .
COPY --from=builder --chown=dataguard:dataguard /build/cmd/data-guard-cli/cli .
COPY --from=builder --chown=dataguard:dataguard /build/cmd/sidecar/sidecar .

USER dataguard

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["./init"]
