FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/deployer ./cmd

FROM alpine:3.21

RUN apk add --no-cache git docker-cli ca-certificates

WORKDIR /app

COPY --from=builder /out/deployer /usr/local/bin/deployer

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/deployer"]
