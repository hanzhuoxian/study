FROM golang:latest AS builder

WORKDIR /github.com/daymenu/user-service

COPY . .

ENV GOPROXY https://goproxy.io

ENV DB_HOST mariadb.host
ENV DB_NAME shipping
ENV DB_USER root
ENV DB_PASSWORD 123456

RUN CGO_ENABLED=0 GOOS=linux go build .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN mkdir /app
WORKDIR /app
COPY --from=builder /github.com/daymenu/user-service .

CMD ["./user-service"]
