FROM alpine:3.24
RUN apk upgrade --no-cache \
    && apk add git

COPY ./build/ /

ENTRYPOINT ["./jx-semanticcheck", "version"]
