FROM golang:1.23 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build the applications
RUN go build -o /app/xrp-indexer ./cmd/indexer/main.go

FROM debian:latest AS execution

WORKDIR /app

COPY --from=builder /app/xrp-indexer .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["./xrp-indexer"]
