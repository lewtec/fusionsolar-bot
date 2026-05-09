FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o fusionsolar-bot ./cmd/fusionsolar-bot

FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/fusionsolar-bot /usr/local/bin/fusionsolar-bot

ENTRYPOINT ["fusionsolar-bot"]
