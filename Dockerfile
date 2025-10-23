FROM golang:1.24 AS builder

WORKDIR /build

COPY go.mod go.sum ./

COPY . ./

# Build the applications
RUN go build -mod=vendor -o /app/xrp-indexer ./cmd/indexer/main.go

RUN git describe --tags --always > PROJECT_VERSION && \
    date --iso-8601=seconds > PROJECT_BUILD_DATE && \
    git rev-parse HEAD > PROJECT_COMMIT_HASH

FROM debian:latest AS execution

WORKDIR /app

COPY --from=builder /app/xrp-indexer .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/PROJECT_VERSION .
COPY --from=builder /build/PROJECT_BUILD_DATE .
COPY --from=builder /build/PROJECT_COMMIT_HASH .

CMD ["./xrp-indexer"]
