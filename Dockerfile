FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o fusionsolar-bot ./cmd/fusionsolar-bot

FROM alpine:latest

RUN apk add --no-cache chromium ca-certificates

COPY --from=builder /app/fusionsolar-bot /usr/local/bin/fusionsolar-bot

ENTRYPOINT ["fusionsolar-bot"]
