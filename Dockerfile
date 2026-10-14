FROM golang:1.24 AS builder
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/gcp-metrics-exporter .

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /out/gcp-metrics-exporter /app/gcp-metrics-exporter
ENTRYPOINT ["/app/gcp-metrics-exporter"]
CMD ["worker", "--config", "/app/config.yaml"]
