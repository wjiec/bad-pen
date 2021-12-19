#!/usr/bin/env bash
set -ex

# export https_proxy=http://172.16.0.x/7890 http_proxy=http://172.16.0.x/7890 all_proxy=http://172.16.0.x/7890
kind create cluster --config=config.yaml
