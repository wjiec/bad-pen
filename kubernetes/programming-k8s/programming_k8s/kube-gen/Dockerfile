FROM golang:alpine as builder

RUN export GOPATH=/go \
    && go install k8s.io/code-generator/cmd/applyconfiguration-gen@latest \
    && go install k8s.io/code-generator/cmd/client-gen@latest \
    && go install k8s.io/code-generator/cmd/conversion-gen@latest \
    && go install k8s.io/code-generator/cmd/deepcopy-gen@latest \
    && go install k8s.io/code-generator/cmd/defaulter-gen@latest \
    && go install k8s.io/code-generator/cmd/informer-gen@latest \
    && go install k8s.io/code-generator/cmd/lister-gen@latest \
    && go install k8s.io/code-generator/cmd/register-gen@latest


FROM golang:alpine

RUN apk add --no-cache bash

COPY --from=builder /go/bin /go/bin
COPY boilerplate.go.txt /hack/boilerplate.go.txt
COPY docker-entrypoint.sh /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
