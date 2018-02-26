FROM        quay.io/prometheus/busybox:latest
MAINTAINER  Robbie Trencheny <me@robbiet.us>

COPY cloudflare_exporter /bin/cloudflare_exporter

EXPOSE     9199
ENTRYPOINT [ "/bin/cloudflare_exporter" ]
