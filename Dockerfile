FROM golang:1.23.4-alpine AS builder
LABEL authors="costi"

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /peer

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o service .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /peer/service .
RUN apk add --no-cache bash
RUN apk add --no-cache nmap

EXPOSE 7777

CMD ["./service"]