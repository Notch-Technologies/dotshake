#!/bin/bash
go build -o dotshake ./cmd/dotshake/dotshake.go
sudo ./dotshake login -signal-host=$SIGNAL_HOST \
    -server-host=$SERVER_HOST \
    -signal-port=$SIGNAL_PORT \
    -server-port=$SERVER_PORT \
    -debug=$IS_DEBUG \
    -loglevel=$LOG_LEVEL

go build -o dotshaker ./cmd/dotshaker/dotshaker.go
sudo ./dotshaker up -signal-host=$SIGNAL_HOST \
    -server-host=$SERVER_HOST \
    -signal-port=$SIGNAL_PORT \
    -server-port=$SERVER_PORT \
    -daemon=$IS_DEAMON \
    -debug=$IS_DEBUG \
    -loglevel=$LOG_LEVEL
