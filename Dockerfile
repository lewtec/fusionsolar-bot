# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/fusionsolar-bot
# No browser in the image — use BROWSER_CDP against a remote CDP endpoint.

FROM alpine:latest@sha256:3f4d6b32e26703c6082839234f43769d5046b3ac7c910033911d1e71d8cd630a

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 65532 nonroot

ARG TARGETPLATFORM
COPY --chown=nonroot:nonroot $TARGETPLATFORM/fusionsolar-bot /usr/local/bin/fusionsolar-bot

USER nonroot

ENTRYPOINT ["fusionsolar-bot"]
