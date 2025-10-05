FROM golang:1.25.1-alpine3.21 AS builder
WORKDIR /app
COPY * ./
RUN go build .

FROM scratch
COPY --from=builder /app/smaps-container-exporter /smaps-container-exporter
ENTRYPOINT ["/smaps-exporter"]
