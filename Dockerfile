FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY . .

RUN go build -o /app ./cmd/app

FROM alpine:latest

COPY --from=builder /app /app
COPY config.yaml /config.yaml

EXPOSE 8080
ENTRYPOINT ["/app", "--config=/config.yaml"]

