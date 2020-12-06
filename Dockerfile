# Build
FROM golang:1.15-alpine as builder

# For compiling dempendancies
RUN apk update && \
    apk upgrade && \
    apk add gcc g++ musl-dev git curl ca-certificates

# Install wait-for for docker-compose
# so that we can wait for the database to
# startup if we need to.
RUN curl -s https://raw.githubusercontent.com/eficode/wait-for/master/wait-for \
    -o /usr/local/bin/wait-for && \
  chmod +x /usr/local/bin/wait-for

# Cache Go dependancies
# Module download will be cached if
# go.mod and go.sum don't change
COPY go.mod /app/tmp/go.mod
COPY go.sum /app/tmp/go.sum
WORKDIR /app/tmp
RUN go mod download

COPY . /app/src
WORKDIR /app/src

RUN CGO_ENABLED=0 go build -o /app/bin/mtupdate ./cmd/mtupdate
RUN CGO_ENABLED=1 go build -o /app/bin/api ./cmd/api

# Run
FROM alpine:latest
# RUN apk add ca-certificates
COPY --from=builder /app/bin /app/bin
COPY --from=builder /usr/local/bin/wait-for /usr/local/bin/wait-for
RUN chmod +x /usr/local/bin/wait-for

CMD ["/app/bin/api", "--port=8080"]
