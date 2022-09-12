#!/bin/bash

function waitservice() {
    HOST=$1
    PORT=$2
    TIMEOUT=$3
    DSN=$4

    if [ "$HOST" == "" ]; then
        echo "Waitservice: No host specified!"
        exit 1;
    fi
    if [ "$PORT" == "" ]; then
        echo "Waitservice: No port specified for $HOST!"
        exit 1;
    fi
    if [ "$TIMEOUT" == "" ]; then
        echo "Waitservice: No timeout specified for $HOST:$PORT!"
        exit 1;
    fi

    if [ -z "$DSN" ]; then
        echo "Waiting $TIMEOUT seconds for $HOST:$PORT"
    fi
    timeout $TIMEOUT bash -c 'until printf "" 2>>/dev/null >>/dev/tcp/$0/$1; do sleep 1; done' $HOST $PORT
    RESULT=$?
    if [ "$RESULT" != "0" ]; then
        echo "Could not establish connection to $HOST:$PORT - $RESULT"
        exit "$RESULT"
    fi
    echo "Connected successfully to $HOST:$PORT"
}

function waitdsn() {
    pattern='^(([[:alnum:]]+)://)?(([[:alnum:]^:]+)@)?([^:^@^/]+)(:([[:digit:]]+))?(\/.*)?$'

    DSN=$1
    DEFAULT_PORT=$2
    TIMEOUT=$3

    if [ "$DSN" == "" ]; then
        echo "Waitservice: No dsn specified!"
        exit 1;
    fi
    if [ "$DEFAULT_PORT" == "" ]; then
        echo "Waitservice: No default port specified!"
        exit 1;
    fi
    if [ "$TIMEOUT" == "" ]; then
        echo "Waitservice: No timeout specified!"
        exit 1;
    fi

    if [[ "$DSN" =~ $pattern ]]; then
        HOST=${BASH_REMATCH[5]}
        PORT=${BASH_REMATCH[7]}
        PORT=${PORT:-$DEFAULT_PORT}
        echo "Waiting $TIMEOUT seconds for $DSN"
        waitservice $HOST $PORT $TIMEOUT $DSN
    else
        echo "Unable to parse $DSN"
    fi
}

#waitdsn $STAN_DSN 4222 ${SERVICE_TIMEOUT:-60}
