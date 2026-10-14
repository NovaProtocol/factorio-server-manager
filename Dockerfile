# syntax=docker/dockerfile:1

# Web UI bundle (webpack + tailwind), output lands in app/.
FROM node:20-alpine AS ui
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY tailwind.config.js webpack.config.js ./
COPY app ./app
COPY ui ./ui
RUN npm run build

# Server binary, cross-compiled for the deployment target.
FROM golang:1.24-alpine AS gobuild
WORKDIR /build/src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/factorio-server-manager .

# Runtime, glibc for the Factorio server binaries.
FROM ubuntu:24.04

ENV FACTORIO_VERSION=stable

VOLUME /opt/fsm-data /opt/factorio/saves /opt/factorio/mods /opt/factorio/config

EXPOSE 80/tcp 34197/udp

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tar xz-utils unzip jq \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt

COPY --from=gobuild /build/factorio-server-manager /opt/fsm/factorio-server-manager
COPY --from=ui /build/app /opt/fsm/app
COPY conf.json.example /opt/fsm/conf.json
COPY docker/entrypoint.sh /opt/entrypoint.sh

ENTRYPOINT ["/opt/entrypoint.sh"]
