FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY pkg/s3/go.mod pkg/s3/
COPY cmd/cli/go.mod cmd/cli/
COPY cmd/data-guard-cli/go.mod cmd/data-guard-cli/

WORKDIR /build/pkg/s3
RUN go mod download

WORKDIR /build/cmd/cli
RUN go mod download

WORKDIR /build/cmd/data-guard-cli
RUN go mod download

COPY pkg/s3/. /build/pkg/s3/
COPY cmd/cli/. /build/cmd/cli/
COPY cmd/data-guard-cli/. /build/cmd/data-guard-cli/

WORKDIR /build/cmd/data-guard-cli
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o data-guard-cli .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /build/cmd/data-guard-cli/data-guard-cli .

ENTRYPOINT ["./data-guard-cli"]
