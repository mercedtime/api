FROM golang:1.15-alpine

COPY . /src
WORKDIR /src
RUN mkdir -p /app/bin

RUN go mod download
RUN go build -o /app/bin/mtupdate ./cmd/mtupdate

RUN rm /etc/crontabs/root
COPY ./db/cron/root /etc/crontabs/root
RUN chown root /etc/crontabs/root
RUN chmod 0600 /etc/crontabs/root

CMD ["crond", "-f", "-l", "0", "-d", "0"]

# vim: ft=dockerfile
