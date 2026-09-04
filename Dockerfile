# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/fusionsolar-bot
# No browser in the image — use BROWSER_CDP against a remote CDP endpoint.

FROM alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

ARG TARGETPLATFORM
COPY --chown=nonroot:nonroot $TARGETPLATFORM/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
