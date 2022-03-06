FROM BASEIMAGE
RUN apk --no-cache add ca-certificates bash

ARG ARCH
ARG TINI_VERSION

ADD cdevents-provider /usr/local/bin/cdevents-provider

EXPOSE 8080
USER 1001
ENTRYPOINT ["cdevents-provider"]
