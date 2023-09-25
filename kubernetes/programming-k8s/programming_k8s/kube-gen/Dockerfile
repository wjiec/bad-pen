FROM golang:alpine

COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN apk add --no-cache git bash \
    && wget -O /etc/bash/kube_codegen.sh https://raw.githubusercontent.com/kubernetes/code-generator/master/kube_codegen.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
