#!/usr/bin/env bash

GO_SRC="$(go env GOPATH)/src"

if [[ -n $CONTROLLER_ROOT ]] && [[ -n $(ls -A "$GO_SRC") ]]; then
  kube::codegen::gen_helpers \
    --boilerplate "$(go env GOPATH)/src/$CONTROLLER_ROOT/hack/boilerplate.go.txt" \
    --input-pkg-root "$CONTROLLER_ROOT/pkg" \
    --output-base "$(go env GOPATH)/src"
  kube::codegen::gen_openapi \
    --input-pkg-root "$CONTROLLER_ROOT/pkg" \
    --output-pkg-root "$CONTROLLER_ROOT/pkg" \
    --output-base "$(go env GOPATH)"
  kube::codegen::gen_client \
    --input-pkg-root "$CONTROLLER_ROOT/pkg" \
    --output-pkg-root "$CONTROLLER_ROOT/pkg" \
    --output-base "$(go env GOPATH)/src" \
    --with-applyconfig \
    --with-watch
else
  exec "$@"
fi
