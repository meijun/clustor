# clustor

[![Build Status](https://travis-ci.org/meijun/clustor.svg?branch=master)](https://travis-ci.org/meijun/clustor)

Simplest cluster monitor

### Download or Build

- Download binary
  - See [releases](https://github.com/meijun/clustor/releases)
- Or build from source:
  - [Install Go](https://golang.org/doc/install)
  - `go get github.com/meijun/clustor`
  - The path of the executable binary is `$GOPATH/bin/clustor`.

### Deployment

You can copy the binary to any host.

- Run as a web server:
  - Copy the binary to the web server host.
  - Listen on a port:
    - `./clustor -listen :7160`
- Run as a worker:
  - Copy the binary to the worker host.
  - Send information to the web server:
    - `./clustor -send http://10.10.7.160:7160`
