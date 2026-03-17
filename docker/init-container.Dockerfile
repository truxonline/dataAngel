# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy module files first (for dependency caching)
COPY cmd/init/go.mod cmd/init/
COPY internal/restore/go.mod internal/restore/
COPY pkg/s3/go.mod pkg/s3/

# Run go mod download in each module directory
WORKDIR /build/internal/restore
RUN go mod download

# Now copy source code
COPY cmd/init/. /build/cmd/init/
COPY internal/restore/. /build/internal/restore/
COPY pkg/s3/. /build/pkg/s3/

# Build inside the module directory (where replace directives work)
WORKDIR /build/cmd/init
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o init-container .

# Final stage - minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 dataguard && \
    adduser -u 1000 -G dataguard -s /bin/sh -D dataguard

WORKDIR /home/dataguard

COPY --from=builder --chown=dataguard:dataguard /build/cmd/init/init-container .

USER dataguard

WORKDIR /home/dataguard
ENTRYPOINT ["./init-container"]
