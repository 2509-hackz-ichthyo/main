FROM golang:1.25.1-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED=1

RUN apk add --no-cache build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/local/bin/ws-decode ./cmd/ws-decode-api

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /usr/local/bin/ws-decode /usr/local/bin/ws-decode

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/ws-decode"]
