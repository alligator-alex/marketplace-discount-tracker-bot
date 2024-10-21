## Build app
FROM golang:1.23.2-alpine AS builder

LABEL version="1.0"

ENV CGO_ENABLED 0
ENV GOOS linux

WORKDIR /slodych/build

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go mod download \
    && go build -ldflags="-s -w" -o ./bot ./cmd/main.go

## Prepare system
FROM alpine:3

RUN apk update --no-cache \
    && apk add --no-cache ca-certificates tzdata logrotate xauth xvfb chromium

COPY ./configs/logrotate.d/bot-logs /etc/logrotate.d/bot-logs
RUN chmod 644 /etc/logrotate.d/bot-logs

## Run
COPY --from=builder /slodych/build/bot /slodych/bot
WORKDIR /slodych
CMD ["./bot"]
