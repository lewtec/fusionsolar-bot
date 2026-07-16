FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Pure Go binary: static link, reproducible paths, smaller artifact.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o fusionsolar-bot ./cmd/fusionsolar-bot

FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

COPY --from=builder --chown=nonroot:nonroot /app/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
