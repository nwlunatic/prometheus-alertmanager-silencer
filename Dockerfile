FROM debian:stretch

ARG IMAGE_VERSION=0.1

RUN  apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY ./bin/silencer /usr/local/bin/silencer

EXPOSE 5000

CMD ["silencer"]
