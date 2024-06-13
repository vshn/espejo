FROM docker.io/library/alpine:3.20 as runtime

ENTRYPOINT ["espejo"]

RUN \
    apk add --no-cache curl bash

COPY espejo /usr/bin/
USER 65532:65532
