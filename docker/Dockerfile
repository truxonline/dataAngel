FROM golang:1.22-alpine AS builder

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

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 dataguard && \
    adduser -u 1000 -G dataguard -s /bin/sh -D dataguard

WORKDIR /home/dataguard

COPY --from=builder --chown=dataguard:dataguard /build/cmd/init/init .
COPY --from=builder --chown=dataguard:dataguard /build/cmd/data-guard-cli/cli .

USER dataguard

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["./init"]
