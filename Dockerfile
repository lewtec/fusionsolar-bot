# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/fusionsolar-bot
# No browser in the image — use BROWSER_CDP against a remote CDP endpoint.

FROM alpine:latest@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

ARG TARGETPLATFORM
COPY --chown=nonroot:nonroot $TARGETPLATFORM/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
