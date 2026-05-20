FROM alpine:3.19

RUN apk add --no-cache \
    mongodb-tools \
    aws-cli \
    bash \
    tzdata

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

# Add crontab file
COPY crontab /var/spool/cron/crontabs/root

CMD ["crond", "-f", "-d", "8"]
