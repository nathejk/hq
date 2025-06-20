ARG GIT_COMMIT="none"
ARG GIT_BRANCH="-"
ARG BUILD_NUMBER="0"
ARG BUILD_VERSION="0"

# DEV - BASE
FROM golang:1.23 AS api-dev

RUN mkdir -p /app/docker/bin
WORKDIR /app

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



# TEST'n'BUILD
## API
FROM api-dev AS api-builder

# copy and download go mod dependencies
COPY go/go.mod go/go.sum ./
RUN go mod download

# copy src
COPY go/ .

#
# BUILD
#FROM base AS build

# build service
RUN GOARCH=amd64 CGO_ENABLED=1 GOOS=linux \
	go build -a -ldflags="-w -s" -o application nathejk.dk/cmd/api


## UI
FROM node:18.9-alpine3.15 AS ui-dev

RUN mkdir -p /app
WORKDIR /app
COPY vue /app
COPY vue/init-dev /init

RUN npm install -g npm@8.19.1

# python is a dependency of node-gyp that we need to build node-sass https://www.npmjs.com/package/node-gyp
RUN apk add g++ make python3 && \
    npm config set python "$(which python3)"

#RUN npm install

ENTRYPOINT ["/init"]

#FROM node:10.11-alpine AS ui-builder
FROM ui-dev AS ui-builder

#npm ci # installs what is specified in package-lock.json
RUN npm ci --no-save
#COPY ui/yarn.lock /app/

#RUN yarn install --frozen-lockfile

#RUN npm test
RUN npm run build

#FROM api-dev AS prod


# PROD
FROM alpine:3.14.1 AS prod

ARG GIT_COMMIT
ARG GIT_BRANCH
ARG BUILD_NUMBER
ARG BUILD_VERSION
ENV GIT_COMMIT=$GIT_COMMIT
ENV GIT_BRANCH=$GIT_BRANCH
ENV BUILD_NUMBER=$BUILD_NUMBER
ENV BUILD_VERSION=$BUILD_VERSION

RUN set -eux; \
	apk add --update --no-cache \
	    bash \
	    coreutils \
	    libc6-compat \
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
