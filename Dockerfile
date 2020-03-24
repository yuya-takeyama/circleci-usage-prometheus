FROM golang:1.14-alpine AS builder

ADD . /app
WORKDIR /app

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"'

FROM alpine:3.11

RUN apk --update add ca-certificates

COPY --from=builder /app/circleci-usage-prometheus /circleci-usage-prometheus

ENTRYPOINT ["/circleci-usage-prometheus"]
