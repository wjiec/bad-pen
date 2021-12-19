#!/usr/bin/env bash
set -ex

openssl genrsa -out server.key 2048
openssl req -new -x509 -key server.key -out server.crt -days 3650 -subj /CN=http-whoami.example.com

# kubectl create secret tls http-whoami-tls --cert=server.crt --key=server.key
