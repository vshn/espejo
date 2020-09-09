
FROM docker.io/library/alpine:3.12 as runtime

ENTRYPOINT ["espejo"]

RUN \
    apk add --no-cache curl bash

COPY espejo /usr/bin/
USER 1000:0
