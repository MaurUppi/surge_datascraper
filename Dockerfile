# Prepare GO build env.
FROM golang:alpine as builder

RUN apk add --no-cache git
WORKDIR /src
COPY scraper-src/. /src
RUN go mod download && \
    go build -o /scraper .

# Built and cp to another docker container. 
FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /scraper .
# cp .env and structure.json
COPY config/. .
ENTRYPOINT ["/app"]