# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/fusionsolar-bot
# No browser in the image — use BROWSER_CDP against a remote CDP endpoint.

FROM alpine:latest@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

ARG TARGETPLATFORM
COPY --chown=nonroot:nonroot $TARGETPLATFORM/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
