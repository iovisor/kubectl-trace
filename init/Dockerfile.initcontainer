FROM alpine:3.8
RUN apk add --update \
  bash \
  bc \
  build-base \
  curl \
  libelf-dev \
  linux-headers \
  make

WORKDIR /

COPY /fetch-linux-headers.sh /

ENTRYPOINT [ "/fetch-linux-headers.sh" ]
