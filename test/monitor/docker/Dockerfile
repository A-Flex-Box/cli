FROM golang:1.24-bookworm

RUN apt-get update && apt-get install -y \
    libpcap-dev \
    conntrack \
    iproute2 \
    net-tools \
    iputils-ping \
    dnsutils \
    curl \
    asciinema \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/cli main.go

RUN mkdir -p /output /recordings

ENV TERM=xterm-256color
ENV COLORTERM=truecolor

COPY test/monitor/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["monitor"]