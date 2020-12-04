FROM golang:1.15-alpine

RUN apk update
RUN apk upgrade
RUN apk add gcc g++ musl-dev curl bash


# Install wait-for-it for docker-compose
# so that we can wait for the database to
# startup if we need to.
RUN curl -s \
  -o /usr/local/bin/wait-for-it.sh \
  https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh
RUN chmod +x /usr/local/bin/wait-for-it.sh

# Compile
COPY . /app/tmp/src
RUN mkdir /app/bin
WORKDIR /app/tmp/src
RUN go mod download
RUN CGO_ENABLED=1 go build -o /app/bin/api ./cmd/api
RUN rm -r /app/tmp

WORKDIR /
EXPOSE 8080

CMD ["/app/bin/api", "--port=8080"]

