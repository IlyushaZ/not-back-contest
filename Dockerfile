FROM golang:1.24 AS builder

COPY . /go/src/repo
WORKDIR /go/src/repo

RUN CGO_ENABLED=0 go build -o ./cmd/server/server ./cmd/server
RUN CGO_ENABLED=0 go build -o ./cmd/items-generator/items-generator ./cmd/items-generator

FROM alpine:latest

# RUN apk add --no-cache cronie

COPY --from=builder /go/src/repo/cmd/server/server .
COPY --from=builder /go/src/repo/cmd/items-generator/items-generator .

COPY --from=builder /go/src/repo/cmd/items-generator/crontab /etc/cron/

RUN chmod +x ./server ./items-generator

RUN crontab /etc/cron/crontab

CMD ["./server"]
