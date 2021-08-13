FROM golang:1.14-alpine as gobuilder

WORKDIR /code

RUN set -ex && \
    apk --no-cache add make git

COPY . .

RUN set -ex && \
    make install

FROM scratch

COPY --from=gobuilder /go/bin/dockfmt /bin/

ENTRYPOINT ["dockfmt"]