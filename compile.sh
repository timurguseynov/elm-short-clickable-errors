#!/usr/bin/env bash

go build -o elm-watch-mac ./watch.go 
GOOS=linux GOARCH=amd64 go build -o elm-watch-linux ./watch.go 
