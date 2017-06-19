# Syndication - An extensible news aggregation server

## Features
* JSON REST API
* Unix socket based Administration API

## Planned Features
* Plugin system
* Automatic HTTPS and SQL database setup

## Building

```
$ mkdir syndication-build && cd syndication-build
$ export GOPATH=$(pwd)
$ mkdir src bin pkg
$ go get github.com/chavamee/syndication
$ cd srg/github.com/chavamee/syndication
$ go build
```
