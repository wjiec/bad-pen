#!/usr/bin/env bash

curl -vvv -H 'Host: http-whoami.example.com' http://localhost/

# *   Trying 127.0.0.1:80...
# * TCP_NODELAY set
# * Connected to localhost (127.0.0.1) port 80 (#0)
# > GET / HTTP/1.1
# > Host: http-whoami.example.com
# > User-Agent: curl/7.68.0
# > Accept: */*
# >
# * Mark bundle as not supporting multiuse
# < HTTP/1.1 200 OK
# < Date: Sun, 19 Dec 2021 14:20:30 GMT
# < Content-Type: text/plain; charset=utf-8
# < Content-Length: 379
# < Connection: keep-alive
# <
# You've hit <http-whoami-rs-dh79t> from "10.244.0.5:51256"
#
# "Accept" => "*/*"
# "User-Agent" => "curl/7.68.0"
# "X-Forwarded-For" => "10.244.0.1"
# "X-Forwarded-Host" => "http-whoami.example.com"
# "X-Forwarded-Port" => "80"
# "X-Forwarded-Proto" => "http"
# "X-Forwarded-Scheme" => "http"
# "X-Real-Ip" => "10.244.0.1"
# "X-Request-Id" => "f996135cb9b2d5f72277309dbb6790b0"
# "X-Scheme" => "http"
# * Connection #0 to host localhost left intact
