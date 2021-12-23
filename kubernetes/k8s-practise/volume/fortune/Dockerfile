FROM alpine:latest

RUN apk add --no-cache fortune
COPY blabber.sh /blabber.sh

ENTRYPOINT ["/blabber.sh"]
