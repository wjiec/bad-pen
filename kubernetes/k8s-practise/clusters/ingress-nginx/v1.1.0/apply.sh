#!/usr/bin/env bash
set -ex

WORKSPACE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}" )" &>/dev/null && pwd -P)

kubectl apply -f "${WORKSPACE}/ingress-nginx.yaml"
