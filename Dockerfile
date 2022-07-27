FROM alpine:3.16
RUN apk upgrade --no-cache \
    && apk add git

COPY ./build/ /

ENTRYPOINT ["./jx-semanticcheck", "version"]
