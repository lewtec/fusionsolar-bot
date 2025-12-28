FROM golang:1.25-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o fusionsolar-bot ./cmd/fusionsolar-bot

FROM alpine:latest

RUN apk add --no-cache chromium ca-certificates

COPY --from=builder /app/fusionsolar-bot /usr/local/bin/fusionsolar-bot

ENTRYPOINT ["fusionsolar-bot"]
