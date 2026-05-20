FROM postgres:16-bookworm
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    daemontools \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN curl -L -o /tmp/wal-g.tar.gz https://github.com/wal-g/wal-g/releases/download/v3.0.8/wal-g-pg-22.04-amd64.tar.gz \
    && tar -zxvf /tmp/wal-g.tar.gz -C /usr/local/bin/ \
    && mv /usr/local/bin/wal-g-pg-22.04-amd64 /usr/local/bin/wal-g \
    && chmod +x /usr/local/bin/wal-g \
    && rm /tmp/wal-g.tar.gz
RUN mkdir -p  /etc/wal-g.env
CMD ["postgres"]    


