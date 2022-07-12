FROM alpine:3.16
RUN apk upgrade --no-cache
RUN apk add git

COPY ./build/ /

ENTRYPOINT ["./jx-semanticcheck", "version"]
