# Base image
FROM alpine:3.8

RUN apk upgrade --update --no-cache
USER nobody

COPY scorecard-proxy /usr/local/bin/scorecard-proxy
