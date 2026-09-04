# checkov:skip=CKV_DOCKER_2: Healthcheck is handled by the container orchestrator
# build executable
FROM golang:1.25.8-trixie@sha256:feb29f1d571353e52350efceeabc6a3f6769c249a97d53cbeececc2ec3f7c202 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build the applications
RUN go build -o /app/xrp-indexer ./cmd/indexer/main.go

RUN git describe --tags --always > PROJECT_VERSION && \
    date --iso-8601=seconds > PROJECT_BUILD_DATE && \
    git rev-parse HEAD > PROJECT_COMMIT_HASH

FROM gcr.io/distroless/cc-debian13:nonroot@sha256:8f960b7fc6a5d6e28bb07f982655925d6206678bd9a6cde2ad00ddb5e2077d78 AS execution

WORKDIR /app
COPY --from=builder /app/xrp-indexer .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/PROJECT_VERSION .
COPY --from=builder /build/PROJECT_BUILD_DATE .
COPY --from=builder /build/PROJECT_COMMIT_HASH .

# numeric UID so k8s runAsNonRoot can verify it
USER 65532:65532

CMD ["./xrp-indexer"]
