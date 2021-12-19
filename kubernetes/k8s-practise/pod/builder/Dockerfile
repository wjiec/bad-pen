FROM golang:alpine as builder

COPY main.go .

RUN go build -o /http-whoami main.go


FROM alpine:latest

COPY --from=builder /http-whoami /usr/local/bin/http-whoami

EXPOSE 8080

ENTRYPOINT ["http-whoami"]
