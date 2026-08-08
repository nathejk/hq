ARG GIT_COMMIT="none"
ARG GIT_BRANCH="-"
ARG BUILD_NUMBER="0"
ARG BUILD_VERSION="0"

# DEV - BASE
FROM golang:1.25 AS api-dev

RUN mkdir -p /app/docker/bin
WORKDIR /app

# Keep the build cache inside /go, which compose mounts as a named volume, so
# tools and dependencies survive restart loops.
ENV GOCACHE=/go/.cache/go-build

ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; \
    apt-get -qq clean && \
	apt-get -qq update && \
    apt-get install -qq -y --no-install-recommends \
        bash \
        inotify-tools && \
    apt-get -qq clean

#RUN printf "#!/bin/sh\necho 'Please mount repo into /app'" > /app/docker/bin/init-dev && \
#    chmod +x /app/docker/bin/init-dev

COPY docker/bin/init-dev /init
ENTRYPOINT ["/init"]
#ENTRYPOINT ["/init"]
CMD ["api"]



# DEV - dependencies pre-downloaded, source copied.
# This is the stage the compose `api` service builds: the module cache baked in
# here seeds the /go volume on first run, so container start does not re-download.
FROM api-dev AS base

# copy and download go mod dependencies
COPY go/go.mod go/go.sum ./
RUN go mod download

# copy src
COPY go/ .


# TEST'n'BUILD
## API
FROM base AS api-builder

# Ignore go.work: CI and prod resolve shared-go from the module proxy at the
# version pinned in go.mod, not from a sibling checkout that does not exist here.
ENV GOWORK=off

#
# BUILD

# Build a fully static, pure-Go binary. CGO is not needed by any dependency, and
# disabling it is what makes the result static — no libc, no libc6-compat shim in
# the runtime image, and the pure-Go DNS resolver instead of glibc's.
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags="-w -s" -o application nathejk.dk/cmd/api


## UI
FROM node:20-alpine AS ui-dev

RUN mkdir -p /app
WORKDIR /app
COPY vue /app
COPY vue/init-dev /init

# Use the npm bundled with Node 20 (v10) — do not re-pin an older npm.
# No build toolchain needed: every dependency is pure JavaScript, so there is no
# node-gyp step to satisfy.

ENTRYPOINT ["/init"]

FROM ui-dev AS ui-builder

ARG GIT_COMMIT="none"
ARG GIT_BRANCH="-"
ARG BUILD_NUMBER="0"

#npm ci # installs what is specified in package-lock.json
RUN npm ci --no-audit --no-fund
RUN npm run build && \
    sed -i "s/__BUILD_GIT_COMMIT__/$GIT_COMMIT/g" dist/index.html && \
    sed -i "s/__BUILD_GIT_BRANCH__/$GIT_BRANCH/g" dist/index.html && \
    sed -i "s/__BUILD_BUILD_NUMBER__/$BUILD_NUMBER/g" dist/index.html

#FROM api-dev AS prod


# PROD
FROM alpine:3.22 AS prod

ARG GIT_COMMIT
ARG GIT_BRANCH
ARG BUILD_NUMBER
ARG BUILD_VERSION
ENV GIT_COMMIT=$GIT_COMMIT
ENV GIT_BRANCH=$GIT_BRANCH
ENV BUILD_NUMBER=$BUILD_NUMBER
ENV BUILD_VERSION=$BUILD_VERSION

# bash: docker/bin/init is a bash script.
# ca-certificates: outbound TLS to CPSMS, MobilePay and the auth service.
# No libc6-compat: the binary is statically linked (see api-builder).
# No tzdata: the timezone database is embedded via time/tzdata in cmd/api.
RUN set -eux; \
	apk add --update --no-cache \
	    bash \
	    coreutils \
	    ca-certificates \
        && \
	rm -rf /tmp/* /var/cache/apk/*

WORKDIR /app
COPY --from=api-builder /app/application /api
COPY --from=ui-builder /app/dist /www
COPY docker/bin/init /init

#HEALTHCHECK --interval=30s --timeout=15s --start-period=900s --retries=3 CMD test -f /tmp/healthy
EXPOSE 80
ENTRYPOINT ["/init"]
CMD ["/api"]
