FROM golang:1.24.5-alpine AS builder
ENV GOTOOLCHAIN=go1.24.5+auto
WORKDIR /app
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags "-s -w" -o presign ./cmd/presign

FROM alpine:3.20 AS runtime
WORKDIR /app
COPY --from=builder /app/presign /usr/local/bin/presign
EXPOSE 8080
CMD ["/usr/local/bin/presign"]
