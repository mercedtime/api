FROM golang:1.15-alpine

COPY . /src
WORKDIR /src
RUN mkdir -p /app/bin

RUN go mod download
RUN go build -o /app/bin/mtupdate ./cmd/mtupdate

WORKDIR /app
RUN rm -rf /src # clean up

RUN rm /etc/crontabs/root
COPY ./db/cron/root /etc/crontabs/root
RUN chown root /etc/crontabs/root
RUN chmod 0600 /etc/crontabs/root

# -d - 0 is verbose, 8 is silent
CMD ["crond", "-f", "-l", "2", "-d", "2"]

# vim: ft=dockerfile
