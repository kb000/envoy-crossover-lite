FROM golang:1.24 AS builder

WORKDIR /workspace

ADD . /workspace

RUN go build -o /usr/local/bin/crossover .

FROM frolvlad/alpine-glibc:alpine-3.22_glibc-2.42

LABEL org.opencontainers.image.base.name="registry.hub.docker.com/mumoshu/crossover"

COPY --from=builder /usr/local/bin/crossover /usr/local/bin
