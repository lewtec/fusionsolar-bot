# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/fusionsolar-bot
# No browser in the image — use BROWSER_CDP against a remote CDP endpoint.

FROM alpine:latest@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

ARG TARGETPLATFORM
COPY --chown=nonroot:nonroot $TARGETPLATFORM/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
