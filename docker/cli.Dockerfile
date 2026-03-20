FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY pkg/s3/go.mod pkg/s3/
COPY cmd/cli/go.mod cmd/cli/
COPY cmd/dataangel-cli/go.mod cmd/dataangel-cli/

WORKDIR /build/pkg/s3
RUN go mod download

WORKDIR /build/cmd/cli
RUN go mod download

WORKDIR /build/cmd/dataangel-cli
RUN go mod download

COPY pkg/s3/. /build/pkg/s3/
COPY cmd/cli/. /build/cmd/cli/
COPY cmd/dataangel-cli/. /build/cmd/dataangel-cli/

WORKDIR /build/cmd/dataangel-cli
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dataangel-cli .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /build/cmd/dataangel-cli/dataangel-cli .

ENTRYPOINT ["./dataangel-cli"]
