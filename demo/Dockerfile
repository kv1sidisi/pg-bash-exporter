FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -o /pg-bash-exporter ./cmd/exporter


FROM alpine:latest

RUN apk add --no-cache bash

COPY --from=builder /pg-bash-exporter /usr/local/bin/pg-bash-exporter

COPY configs/config.example.yaml /etc/pg-bash-exporter/config.yaml

ENTRYPOINT [ "pg-bash-exporter" ]

CMD [ "--config", "/etc/pg-bash-exporter/config.yaml" ]