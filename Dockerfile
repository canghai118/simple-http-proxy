FROM alpine

ADD simple-http-proxy /usr/local/bin

ENTRYPOINT ["simple-http-proxy"]